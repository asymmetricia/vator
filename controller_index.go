package main

import (
	"fmt"
	"net/http"

	"github.com/jrmycanady/nokiahealth"
	"github.com/pdbogen/vator/models"
	"go.etcd.io/bbolt"
)

func IndexHandler(db *bbolt.DB, withings *nokiahealth.Client) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		if user.RefreshSecret == "" {
			WithingsBeginOauth(db, withings, rw, req)
		} else {
			ctx, err := notifications(db, req)
			if err != nil {
				Bail(rw, req, err, http.StatusInternalServerError)
				return
			}
			ctx.Phone = user.Phone
			ctx.Kgs = user.Kgs

			TemplateGet(rw, req, "index.tmpl", ctx)
		}
	}
}
