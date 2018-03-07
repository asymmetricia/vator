package main

import (
	"fmt"
	"github.com/coreos/bbolt"
	"github.com/pdbogen/nokiahealth"
	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"net/http"
	"strconv"
)

func OauthHandler(db *bolt.DB, nokia nokiahealth.Client, consumerSecret string) func(http.ResponseWriter, *http.Request) {
	return RequireForm([]string{"oauth_token", "oauth_verifier", "userid"}, func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("user should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}

		requestToken, err := models.SessionGet(db, req, "request_token")
		if err != nil {
			Bail(rw, req, fmt.Errorf("retrieving request_token: %s", err), http.StatusInternalServerError)
			return
		}
		requestSecret, err := models.SessionGet(db, req, "request_secret")
		if err != nil {
			Bail(rw, req, fmt.Errorf("retrieving request_secret: %s", err), http.StatusInternalServerError)
			return
		}

		if requestToken == "" || requestSecret == "" {
			http.Redirect(rw, req, "/", http.StatusFound)
			return
		}

		userid, err := strconv.Atoi(req.Form.Get("userid"))
		verifier := req.Form.Get("oauth_verifier")
		if err != nil {
			http.Error(rw, "I had trouble parsing that userid.", http.StatusBadRequest)
			Log.Errorf("failed to parse userid %q: %s", req.Form.Get("userid"), err)
			return
		}
		ar := nokia.RebuildAccessRequest(requestToken, requestSecret)
		oauthUser, err := ar.GenerateUser(verifier, userid)
		if err != nil {
			Bail(rw, req, fmt.Errorf("obtaining user with oauth verifier: %s", err), http.StatusInternalServerError)
			return
		}

		user.Id = oauthUser.UserID
		user.Secret = oauthUser.AccessSecretStr
		user.Token = oauthUser.AccessTokenStr
		if err := user.Save(db); err != nil {
			Bail(rw, req, fmt.Errorf("saving user: %s", err), http.StatusInternalServerError)
			return
		}
		http.Redirect(rw, req, "/", http.StatusFound)
	})
}

func BeginOauth(db *bolt.DB, nokia nokiahealth.Client, rw http.ResponseWriter, req *http.Request) {
	ar, err := nokia.CreateAccessRequest()
	if err != nil {
		Bail(rw, req, fmt.Errorf("starting sign-up process: %s", err), http.StatusInternalServerError)
		return
	}
	if err := models.SessionSet(db, req, "request_token", ar.RequestToken); err != nil {
		Bail(rw, req, fmt.Errorf("saving request_token: %s", err), http.StatusInternalServerError)
		return
	}
	if err := models.SessionSet(db, req, "request_secret", ar.RequestSecret); err != nil {
		Bail(rw, req, fmt.Errorf("saving request_secret: %s", err), http.StatusInternalServerError)
		return
	}

	StaticGet(rw, req, fmt.Sprintf("Welcome to vator! Click <a href='%s'>here</a> to link up to your Nokia Health account.", ar.AuthorizationURL))
}
