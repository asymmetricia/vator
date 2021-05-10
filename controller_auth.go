package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

func Bail(rw http.ResponseWriter, req *http.Request, err error, status int) {
	Log.ExtraCalldepth++
	Log.Errorf("%s: %s", req.RemoteAddr, err)
	Log.ExtraCalldepth--
	http.Error(rw, "very sorry! afraid something went wrong...", status)
	return
}

func RequireAuth(db *bbolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.SessionGet(db, req, "user")
		if err != nil {
			rw.Header().Add("location", "/login")
			http.Error(rw, "You are not logged in.", http.StatusFound)
			return
		}
		handler(rw, req.WithContext(context.WithValue(req.Context(), "user", user)))
	}
}

func RequireNotAuth(db *bbolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		_, err := models.SessionGet(db, req, "user")
		if err == nil {
			rw.Header().Add("location", "/")
			http.Error(rw, "You are logged in!", http.StatusFound)
			return
		}
		handler(rw, req)
	}
}

func notifications(db *bbolt.DB, req *http.Request) (TemplateContext, error) {
	ctx := TemplateContext{}
	for key, dest := range map[string]*string{"error": &ctx.Error, "toast": &ctx.Toast} {
		value, err := models.SessionGet(db, req, key)
		if err == models.KeyDoesNotExist {
			err = nil
		}
		if err != nil {
			return TemplateContext{}, fmt.Errorf("getting %s msg from session: %s", key, err)
		}
		*dest = value
		err = models.SessionSet(db, req, key, "")
		if err != nil {
			return TemplateContext{}, fmt.Errorf("clearing %s msg from session: %s", key, err)
		}
	}
	return ctx, nil
}

func LoginHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "POST":
			RequireForm([]string{"username", "password"}, LoginHandlerPost(db))(rw, req)
		default:
			TemplateGet(rw, req, "login.tmpl", TemplateContext{Page: "login"})
		}
	}
}

func LoginHandlerPost(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		invalid := func() {
			http.Error(rw, "The username or password you provided was invalid.", http.StatusBadRequest)
		}
		user, err := models.LoadUser(db, req.Form.Get("username"))
		if errors.Is(err, models.UserNotFound) {
			invalid()
			return
		}
		if err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}
		if err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(req.Form.Get("password"))); err != nil {
			invalid()
			return
		}
		if err := models.SessionSet(db, req, "user", user.Username); err != nil {
			Bail(rw, req, err, http.StatusInternalServerError)
			return
		}
		http.Redirect(rw, req, "/", http.StatusFound)
	}
}

func TemplateGet(rw http.ResponseWriter, _ *http.Request, template string, ctx TemplateContext) {
	err := templates.ExecuteTemplate(rw, template, ctx)
	if err != nil {
		Log.Errorf("error rendering template: %v", err)
		http.Error(rw, "Very sorry; something went wrong.", http.StatusInternalServerError)
		return
	}
}

func LogoutHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		http.SetCookie(rw, &http.Cookie{Name: "session", Expires: time.Unix(0, 0)})
		if err := models.SessionDeleteReq(db, req); err != nil {
			Log.Errorf("error rendering template: %v", err)
			http.Error(rw, "Very sorry; something went wrong.", http.StatusInternalServerError)
			return
		}

		http.Redirect(rw, req, "/login", http.StatusFound)
	}
}

func PhoneHandler(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "POST":
			RequireForm([]string{"phone"}, PhoneHandlerPost(db))(rw, req)
		default:
			http.Redirect(rw, req, "/", http.StatusFound)
		}
	}
}

func PhoneHandlerPost(db *bbolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		user.Phone = req.FormValue("phone")
		if err := user.Save(db); err != nil {
			Bail(rw, req, fmt.Errorf("saving user %q: %s", user.Username, err), http.StatusInternalServerError)
			return
		}
		err = models.SessionSet(db, req, "toast", "phone number updated!")
		if err != nil {
			Bail(rw, req, fmt.Errorf("setting toast msg in session: %s", err), http.StatusInternalServerError)
			return
		}
		http.Redirect(rw, req, "/", http.StatusFound)
	}
}
