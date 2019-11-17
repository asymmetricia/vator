package main

import (
	"fmt"
	"github.com/coreos/bbolt"
	"github.com/pdbogen/vator/models"
	"net/http"
)

func SummaryHandler(db *bbolt.DB, twilio *models.Twilio) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		u, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return

		}

		u.Summary(twilio, db, true)

		err = models.SessionSet(db, req, "toast", "summary is on its way!")
		if err != nil {
			Bail(rw, req, fmt.Errorf("setting toast msg in session: %s", err), http.StatusInternalServerError)
			return
		}
		http.Redirect(rw, req, "/", http.StatusFound)
	}
}
