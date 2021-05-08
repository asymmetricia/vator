package main

import (
	"fmt"
	"net/http"

	"github.com/jrmycanady/nokiahealth"
	"github.com/pdbogen/vator/models"
	"go.etcd.io/bbolt"
)

type WithingsClient struct {
	Db       *bbolt.DB
	Withings *nokiahealth.Client
}

func (w *WithingsClient) Begin(rw http.ResponseWriter, req *http.Request) {
	url, state, err := w.Withings.AuthCodeURL()
	if err != nil {
		Bail(rw, req, fmt.Errorf("generating authorization URL: %s", err), http.StatusInternalServerError)
		return
	}

	if err := SaveState(w.Db, state); err != nil {
		Bail(rw, req, fmt.Errorf("saving generated state: %s", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, req, url, http.StatusFound)
}

func (w *WithingsClient) Complete(rw http.ResponseWriter, req *http.Request) {
	RequireForm([]string{"code", "state"}, func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(w.Db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("loading user from db for request: %s", err), http.StatusInternalServerError)
			return
		}

		err = ConsumeState(w.Db, req.Form.Get("state"))

		if err != nil {
			Bail(rw, req, fmt.Errorf("state %q: %s", req.Form.Get("state"), err), http.StatusBadRequest)
			return
		}

		withingsUser, err := w.Withings.NewUserFromAuthCode(req.Context(), req.Form.Get("code"))
		if err != nil {
			Bail(rw, req, fmt.Errorf("geting user from auth code %q: %s", req.Form.Get("code"), err), http.StatusBadRequest)
			return
		}

		token := withingsUser.OauthToken

		user.AccessToken = token.AccessToken
		user.RefreshSecret = token.RefreshToken
		user.TokenExpiry = token.Expiry
		if err := user.Save(w.Db); err != nil {
			Bail(rw, req, fmt.Errorf("saving user: %s", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(rw, req, "/", http.StatusFound)
	})(rw, req)
}
