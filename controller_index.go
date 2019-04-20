package main

import (
	"fmt"
	"github.com/coreos/bbolt"
	"github.com/jrmycanady/nokiahealth"
	"github.com/pdbogen/vator/models"
	"net/http"
)

func IndexHandler(db *bbolt.DB, withings nokiahealth.Client) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		if user.RefreshSecret == "" {
			BeginOauth(db, withings, rw, req)
		} else {
			ctx, err := notifications(db, req)
			if err != nil {
				Bail(rw, req, err, http.StatusInternalServerError)
				return
			}
			ctx["phone"] = user.Phone
			if user.Kgs {
				ctx["kgs"] = "true"
			}

			TemplateGet(rw, req, indexTemplate, ctx)
		}
	}
}
