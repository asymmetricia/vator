package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/asymmetricia/nokiahealth"
	. "github.com/asymmetricia/vator/log"
	"github.com/asymmetricia/vator/models"
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

var minBackfill = time.Date(2008, time.January, 0, 0, 0, 0, 0, time.UTC)

func BackfillMeasures(db *bbolt.DB, withings *nokiahealth.Client) {
	for _, u := range models.GetUsers(db) {
		if u.BackFillDate.IsZero() {
			Log.Debugf("initializing backfill for %q", u.Username)
			u.BackFillDate = time.Now()
		}

		if u.BackFillDate.Before(minBackfill) {
			Log.Debugf("backfill complete for %q -> %s", u.Username, u.BackFillDate)
			continue
		}

		before := len(u.Weights)
		bfFrom := u.BackFillDate.Add(-365 * 24 * time.Hour)
		bfTo := u.BackFillDate

		err := u.GetWeights(db, withings, bfFrom, bfTo)

		if err == nil {
			u.BackFillDate = bfFrom
			err = u.Save(db)
		}

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
			Log.Debugf("%q: %d weights before update, %d after; sending toast",
				u.Username, before, len(u.Weights))
			go u.Toast(twilio)
		} else {
			Log.Debugf("no new weights for %q", u.Username)
		}
		go u.Summary(twilio, db, false)
	}
}
