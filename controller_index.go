package main

import (
	"fmt"
	"net/http"

	"github.com/asymmetricia/vator/models"
	"github.com/asymmetricia/withings"
	"go.etcd.io/bbolt"
)

func IndexHandler(db *bbolt.DB, withings *withings.Client) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("in IndexHandler: should be logged in, "+
				"but: %s", err), http.StatusInternalServerError)
			return
		}

		ctx, err := notifications(db, req)
		if err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}
		ctx.Phone = user.Phone
		ctx.Kgs = user.Kgs
		ctx.Share = user.Share
		ctx.Withings = user.RefreshSecret != ""

		ctx.Page = "index"
		ctx.User = user.Username

		TemplateGet(rw, req, "index.tmpl", ctx)
	}
}
