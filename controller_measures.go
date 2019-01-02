package main

import (
	"fmt"
	"github.com/coreos/bbolt"
	"github.com/jrmycanady/nokiahealth"
	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"net/http"
	"time"
)

func MeasuresHandler(db *bbolt.DB, nokia nokiahealth.Client) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		u, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return

		}

		weights, err := u.GetWeights(nokia)
		if err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}
		if len(weights) > 5 {
			weights = weights[0:5]
		}
		for i := len(weights)-1; i >= 0; i-- {
			w := weights[i]
			fmt.Fprintln(rw, w.Date, " ", u.FormatKg(w.Kgs))
		}
	}
}

func ScanMeasures(db *bbolt.DB, nokia nokiahealth.Client, twilio *models.Twilio) {
	for _, u := range models.GetUsers(db) {
		if u.LastWeight.IsZero() {
			u.LastWeight = time.Now().AddDate(0, 0, -200)
		}
		weights, err := u.GetWeightsSince(nokia, u.LastWeight.Add(time.Minute))
		if err != nil {
			Log.Warning("error getting weights for %q: %s", u.Username, err)
			continue
		}
		if len(weights) == 0 {
			Log.Debugf("no new weights for %q", u.Username)
			continue
		}
		for _, w := range weights {
			if w.Date.After(u.LastWeight) {
				u.LastWeight = w.Date
			}
			u.Weights = append(u.Weights, models.Weight{w.Date, w.Kgs})
		}
		if err := u.Save(db); err != nil {
			Log.Warning("saving user %q after logging weights: %s", u.Username, err)
			continue
		}
		go u.Toast(twilio)
		Log.Infof("logged %d new measurement(s) for user %q", len(weights), u.Username)
	}
}
