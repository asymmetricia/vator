package main

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/jrmycanady/nokiahealth"
	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"go.etcd.io/bbolt"
)

func MeasuresHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		u, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return

		}

		for _, w := range u.Weights {
			if w.Date.Before(time.Now().Add(-14 * 24 * time.Hour)) {
				continue
			}
			if _, err := fmt.Fprintln(rw, w.Date, " ", u.FormatKg(w.Kgs)); err != nil {
				Log.Errorf("writing output to user: %s", err)
				return
			}
		}
	}
}

func ScanMeasures(db *bbolt.DB, withings *nokiahealth.Client, twilio *models.Twilio) {
	for _, u := range models.GetUsers(db) {
		if u.LastWeight.IsZero() {
			u.LastWeight = time.Now().AddDate(0, 0, -37)
		}
		weights, err := u.GetWeightsSince(db, withings, u.LastWeight.Add(time.Minute))
		if err != nil {
			Log.Warningf("error getting weights for %q: %s", u.Username, err)
			continue
		}

		for _, w := range weights {
			if w.Date.After(u.LastWeight) {
				u.LastWeight = w.Date
			}
			u.Weights = append(u.Weights, models.Weight{w.Date, w.Kgs})
		}

		if len(weights) > 0 {
			sort.Slice(u.Weights, func(i, j int) bool { return u.Weights[i].Date.Before(u.Weights[j].Date) })

			if err := u.Save(db); err != nil {
				Log.Warning("saving user %q after logging weights: %s", u.Username, err)
				continue
			}
			Log.Infof("logged %d new measurement(s) for user %q", len(weights), u.Username)
			go u.Toast(twilio)
		} else {
			Log.Debugf("no new weights for %q", u.Username)
		}
		go u.Summary(twilio, db, false)
	}
}
