package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreos/bbolt"
	"github.com/jrmycanady/nokiahealth"
	"github.com/pdbogen/vator/models"
	"net/http"
	"time"
)

const StatesBucket = "states"

func OauthHandler(db *bbolt.DB, nokia nokiahealth.Client) func(http.ResponseWriter, *http.Request) {
	return RequireForm([]string{"code", "state"}, func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("loading user from db for request: %s", err), http.StatusInternalServerError)
			return
		}

		err = db.Update(func(tx *bbolt.Tx) error {
			states := tx.Bucket([]byte(StatesBucket))
			if states == nil {
				return errors.New("state bucket not found")
			}

			state := states.Get([]byte(req.Form.Get("state")))
			if state == nil {
				return errors.New("state entry not found")
			}

			var expiry time.Time
			if err := expiry.UnmarshalText(state); err != nil {
				return fmt.Errorf("state expiry %q not valid", string(state))
			}

			if expiry.Before(time.Now()) {
				return fmt.Errorf("state expired at %s", expiry.Format(time.RFC1123Z))
			}

			return nil
		})
		if err != nil {
			Bail(rw, req, fmt.Errorf("state %q: %s", req.Form.Get("state"), err), http.StatusBadRequest)
			return
		}

		nokiaUser, err := nokia.NewUserFromAuthCode(context.Background(), req.Form.Get("code"))
		if err != nil {
			Bail(rw, req, fmt.Errorf("geting user from auth code %q: %s", req.Form.Get("code"), err), http.StatusBadRequest)
			return
		}

		token, err := nokiaUser.Token.Token()
		if err != nil {
			Bail(rw, req, fmt.Errorf("getting token from withings user: %s", err), http.StatusBadRequest)
			return
		}

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

func SaveState(db *bbolt.DB, state string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(StatesBucket))
		if err != nil {
			return fmt.Errorf("getting `%s` bucket: %s", StatesBucket, err)
		}

		var deletions [][]byte
		err = bucket.ForEach(func(k, v []byte) error {
			var expiry time.Time
			if err := expiry.UnmarshalText(v); err != nil {
				deletions = append(deletions, k)
				return nil
			}
			if expiry.Before(time.Now()) {
				deletions = append(deletions, k)
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
		for _, del := range deletions {
			if err := bucket.Delete(del); err != nil {
				panic(err)
			}
		}

		expiry, err := time.Now().Add(time.Hour).MarshalText()
		if err != nil {
			panic(err)
		}
		return bucket.Put([]byte(state), expiry)
	})
}

func BeginOauth(db *bbolt.DB, nokia nokiahealth.Client, rw http.ResponseWriter, req *http.Request) {
	url, state, err := nokia.AuthCodeURL()
	if err != nil {
		Bail(rw, req, fmt.Errorf("generating authorization URL: %s", err), http.StatusInternalServerError)
		return
	}

	if err := SaveState(db, state); err != nil {
		Bail(rw, req, fmt.Errorf("saving generated state: %s", err), http.StatusInternalServerError)
		return
	}

	StaticGet(rw, req, fmt.Sprintf("Welcome to vator! Click <a href='%s'>here</a> to link up to your Nokia Health account.", url))
}

func ReauthHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		user.RefreshSecret = ""
		user.AccessToken = ""
		user.TokenExpiry = time.Time{}

		if err := user.Save(db); err != nil {
			Bail(rw, req, fmt.Errorf("saving user: %s", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(rw, req, "/", http.StatusFound)
	}
}
