package main

import (
	"fmt"
	"github.com/coreos/bbolt"
	"github.com/pdbogen/nokiahealth"
	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"net/http"
	"strconv"
	"time"
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
		Log.Errorf("got callback (token=%q, verifier=%q, userid=%q)",
			req.Form.Get("oauth_token"),
			req.Form.Get("oauth_verifier"),
			req.Form.Get("userid"),
		)
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
	var arUrl, expiry string
	data, err := models.SessionGetMulti(db, req, []string{"oauth_expiry", "authorization_url"})

	if err == nil {
		expiry = data[0]
		arUrl = data[1]
		var t time.Time
		expErr := t.UnmarshalText([]byte(expiry))
		if expErr != nil || time.Now().After(t) {
			// if the data from the session is expired, pretend there was no data at all
			err = models.KeyDoesNotExist
		}
	}
	// data is either missing or something bad happened retrieving it
	if err != nil {
		if err != models.KeyDoesNotExist {
			// something bad happened, so bail
			Bail(rw, req, fmt.Errorf("obtaining data from session: %s", err), http.StatusInternalServerError)
			return
		}
		// data is missing, so create new data and save it
		ar, err := nokia.CreateAccessRequest()
		if err != nil {
			Bail(rw, req, fmt.Errorf("starting sign-up process: %s", err), http.StatusInternalServerError)
			return
		}
		expiryBytes, err := time.Now().Add(time.Hour).MarshalText()
		if err != nil {
			Bail(rw, req, fmt.Errorf("rendering time to text: %s", err), http.StatusInternalServerError)
			return
		}
		expiry = string(expiryBytes)
		arUrl = ar.AuthorizationURL.String()
		err = models.SessionSetMulti(db, req,
			[]string{"oauth_expiry", "authorization_url", "request_token", "request_secret"},
			[]string{expiry, arUrl, ar.RequestToken, ar.RequestSecret},
		)
		if err != nil {
			Bail(rw, req, fmt.Errorf("saving data to session: %s", err), http.StatusInternalServerError)
			return
		}
	}

	StaticGet(rw, req, fmt.Sprintf("Welcome to vator! Click <a href='%s'>here</a> to link up to your Nokia Health account.", arUrl))
}

func ReauthHandler(db *bolt.DB, nokia nokiahealth.Client) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		user.Id = 0

		if err := user.Save(db); err != nil {
			Bail(rw, req, fmt.Errorf("saving user: %s", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(rw, req, "/", http.StatusFound)
	}
}
