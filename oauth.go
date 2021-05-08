package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jrmycanady/nokiahealth"
	"github.com/pdbogen/vator/models"
	"go.etcd.io/bbolt"
)

const StatesBucket = "states"

func WithingsOauthHandler(db *bbolt.DB, withings *nokiahealth.Client) func(http.ResponseWriter, *http.Request) {
	return RequireForm([]string{"code", "state"}, func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("loading user from db for request: %s", err), http.StatusInternalServerError)
			return
		}

		err = ConsumeState(db, req.Form.Get("state"))

		if err != nil {
			Bail(rw, req, fmt.Errorf("state %q: %s", req.Form.Get("state"), err), http.StatusBadRequest)
			return
		}

		withingsUser, err := withings.NewUserFromAuthCode(context.Background(), req.Form.Get("code"))
		if err != nil {
			Bail(rw, req, fmt.Errorf("geting user from auth code %q: %s", req.Form.Get("code"), err), http.StatusBadRequest)
			return
		}

		token := withingsUser.OauthToken

		user.AccessToken = token.AccessToken
		user.RefreshSecret = token.RefreshToken
		user.TokenExpiry = token.Expiry
		if err := user.Save(db); err != nil {
			Bail(rw, req, fmt.Errorf("saving user: %s", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(rw, req, "/", http.StatusFound)
	})
}

func WithingsBeginOauth(db *bbolt.DB, withings *nokiahealth.Client, rw http.ResponseWriter, req *http.Request) {
	url, state, err := withings.AuthCodeURL()
	if err != nil {
		Bail(rw, req, fmt.Errorf("generating authorization URL: %s", err), http.StatusInternalServerError)
		return
	}

	if err := SaveState(db, state); err != nil {
		Bail(rw, req, fmt.Errorf("saving generated state: %s", err), http.StatusInternalServerError)
		return
	}

	StaticGet(rw, req, fmt.Sprintf("Welcome to vator! Click <a href='%s'>here</a> to link up to your Withings account.", url))
}

func WithingsReauthHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		user.RefreshSecret = ""
		user.AccessToken = ""

		if err := user.Save(db); err != nil {
			Bail(rw, req, fmt.Errorf("saving user: %s", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(rw, req, "/", http.StatusFound)
	}
}
