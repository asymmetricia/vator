package main

import (
	"github.com/asymmetricia/vator/models"
	"go.etcd.io/bbolt"

	"net/http"
)

func KgsHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}

		if err := req.ParseForm(); err != nil {
			Bail(rw, req, err, http.StatusBadRequest)
			return
		}

		user.Kgs = !user.Kgs
		if err := user.Save(db); err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}

		http.Redirect(rw, req, "/", http.StatusFound)
	}
}
