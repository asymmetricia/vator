package main

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/asymmetricia/vator/models"
	"go.etcd.io/bbolt"
)

func RenameHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}

		switch req.Method {
		case http.MethodPost:
			RenameHandlerPost(db, user, rw, req)
		default:
			errMsg, _ := models.SessionGet(db, req, "error")
			_ = models.SessionSet(db, req, "error", "")
			TemplateGet(rw, req, "rename.tmpl", TemplateContext{
				Page:  "rename",
				Error: errMsg,
				User:  user.Username,
			})
		}
	}
}
func RenameHandlerPost(db *bbolt.DB, user *models.User, rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		Bail(rw, req, errors.New("POST required"), http.StatusBadRequest)
		return
	}

	if err := req.ParseForm(); err != nil {
		Bail(rw, req, err, http.StatusBadRequest)
		return
	}

	newUsername := req.Form.Get("new_name")
	if _, err := models.LoadUser(db, newUsername); !errors.Is(err, models.UserNotFound) {
		if err := models.SessionSet(db, req, "error", "that username is taken!"); err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}
		http.Redirect(rw, req, "/rename", http.StatusFound)
		return
	}

	deadname := user.Username
	err := user.Rename(db, newUsername)
	if err == nil {
		err = models.SessionSetMulti(db, req, []string{"user", "toast"},
			[]string{strings.ToLower(newUsername), "I love it!!! Saved!"})
	}
	if err != nil {
		Bail(rw, req, err, http.StatusInternalServerError)
		return
	}

	log.Printf("user %q renamed to %q", deadname, newUsername)

	http.Redirect(rw, req, "/", http.StatusFound)
}
