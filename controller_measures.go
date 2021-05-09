package main

import (
	"fmt"
	"net/http"
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

		u.WeightsMu.RLock()
		defer u.WeightsMu.RUnlock()
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

var minBackfill = time.Date(2008, time.January, 0, 0, 0, 0, 0, time.UTC)

func BackfillMeasures(db *bbolt.DB, withings *nokiahealth.Client) {
	for _, u := range models.GetUsers(db) {
		if u.BackFillDate.Before(minBackfill) {
			continue
		}

		if u.BackFillDate.IsZero() {
			u.BackFillDate = time.Now()
		}

		before := len(u.Weights)
		bfd := u.BackFillDate
		u.BackFillDate = u.BackFillDate.Add(-30 * 24 * time.Hour)
		err := u.GetWeights(db, withings, u.BackFillDate, bfd)
		if err != nil {
			Log.Warningf("error backfilling weights for %q: %s", u.Username, err)
			return
		}
		if before != len(u.Weights) {
			Log.Debugf("fetched %d old weights for %q", len(u.Weights)-before,
				u.Username)
		}
	}
}

func ScanMeasures(db *bbolt.DB, withings *nokiahealth.Client, twilio *models.Twilio) {
	for _, u := range models.GetUsers(db) {
		if u.LastWeight.IsZero() {
			u.LastWeight = time.Now().AddDate(0, 0, -37)
		}

		before := len(u.Weights)
		err := u.GetWeights(db, withings, u.LastWeight.Add(time.Minute), time.Now())
		if err != nil {
			Log.Warningf("error getting weights for %q: %s", u.Username, err)
			continue
		}

		if len(u.Weights) != before {
			go u.Toast(twilio)
		} else {
			Log.Debugf("no new weights for %q", u.Username)
		}
		go u.Summary(twilio, db, false)
	}
}
