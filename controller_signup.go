package main

import (
	"errors"
	"net/http"

	"github.com/asymmetricia/vator/models"
	"go.etcd.io/bbolt"
)

func SignupHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "POST":
			RequireForm([]string{"username", "password", "confirm"}, SignupHandlerPost(db))(rw, req)
		default:
			SignupHandlerGet(db, rw, req)
		}
	}
}

func SignupHandlerGet(db *bbolt.DB, rw http.ResponseWriter, req *http.Request) {
	notif, err := notifications(db, req)
	if err != nil {
		Bail(rw, req, err, http.StatusInternalServerError)
		return
	}
	notif.Page = "signup"
	TemplateGet(rw, req, "signup.tmpl", notif)
}

func SignupHandlerPost(db *bbolt.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		username := req.Form.Get("username")
		password := req.Form.Get("password")
		confirm := req.Form.Get("confirm")
		_, err := models.LoadUser(db, username)
		if !errors.Is(err, models.UserNotFound) {
			err = errors.New("that username is taken; try again?")
		} else {
			err = nil
		}
		if err == nil && password != confirm {
			err = errors.New("your passwords did not match; give it another shot?")
		}

		if err != nil {
			models.SessionSet(db, req, "error", err.Error())
			http.Redirect(rw, req, "/signup", http.StatusFound)
			return
		}

		u := &models.User{Username: username}
		if err := u.SetPassword(password); err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}

		if err := u.Save(db); err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}

		if err := models.SessionSet(db, req, "user", username); err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}

		http.Redirect(rw, req, "/", http.StatusFound)
		return

	}
}
