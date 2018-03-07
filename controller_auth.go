package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/cbroglie/mustache"
	"github.com/coreos/bbolt"
	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

func Bail(rw http.ResponseWriter, req *http.Request, err error, status int) {
	Log.Errorf("%s: %s", req.RemoteAddr, err)
	http.Error(rw, "very sorry! afraid something went wrong...", status)
	return
}

func RequireAuth(db *bolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
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

func RequireNotAuth(db *bolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
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

func SignupHandler(db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "POST":
			RequireForm([]string{"username", "password", "confirm"}, SignupHandlerPost(db))(rw, req)
		default:
			SignupHandlerGet(db, rw, req)
		}
	}
}

func notifications(db *bolt.DB, req *http.Request) (map[string]string, error) {
	ctx := map[string]string{}
	for _, key := range []string{"error", "toast"} {
		value, err := models.SessionGet(db, req, key)
		if err == models.KeyDoesNotExist {
			err = nil
		}
		if err != nil {
			return nil, fmt.Errorf("getting %s msg from session: %s", key, err)
		}
		ctx[key] = value
		err = models.SessionSet(db, req, key, "")
		if err != nil {
			return nil, fmt.Errorf("clearing %smsg from session: %s", key, err)
		}
	}
	return ctx, nil
}

func SignupHandlerGet(db *bolt.DB, rw http.ResponseWriter, req *http.Request) {
	notif, err := notifications(db, req)
	if err != nil {
		Bail(rw, req, err, http.StatusInternalServerError)
		return
	}
	TemplateGet(rw, req, signupTemplate, notif)
}

func SignupHandlerPost(db *bolt.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		username := req.Form.Get("username")
		password := req.Form.Get("password")
		confirm := req.Form.Get("confirm")
		_, err := models.LoadUser(db, username)
		if err != models.UserNotFound {
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

func LoginHandler(db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "POST":
			RequireForm([]string{"username", "password"}, LoginHandlerPost(db))(rw, req)
		default:
			StaticGet(rw, req, loginHtml)
		}
	}
}

func LoginHandlerPost(db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		invalid := func() {
			http.Error(rw, "The username or password you provided was invalid.", http.StatusBadRequest)
		}
		user, err := models.LoadUser(db, req.Form.Get("username"))
		if err == models.UserNotFound {
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

func StaticGet(rw http.ResponseWriter, _ *http.Request, content string) {
	rw.Header().Add("content-type", "text/html; charset=utf-8")
	fmt.Fprint(rw, content)
}

func TemplateGet(rw http.ResponseWriter, _ *http.Request, template string, context ...interface{}) {
	out, err := mustache.RenderPartials(template, partials, context...)
	if err != nil {
		Log.Error("error rendering template: %s", err)
		http.Error(rw, "Very sorry; something went wrong.", http.StatusInternalServerError)
		return
	}
	rw.Header().Add("content-type", "text/html; charset=utf-8")
	fmt.Fprintf(rw, out)
}

func LogoutHandler(db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		http.SetCookie(rw, &http.Cookie{Name: "session", Expires: time.Unix(0, 0)})
		if err := models.SessionDelete(db, req); err != nil {
			Log.Error("error rendering template: %s", err)
			http.Error(rw, "Very sorry; something went wrong.", http.StatusInternalServerError)
			return
		}
		http.Redirect(rw, req, "/login", http.StatusFound)
	}
}

func PhoneHandler(db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "POST":
			RequireForm([]string{"phone"}, PhoneHandlerPost(db))(rw, req)
		default:
			http.Redirect(rw, req, "/", http.StatusFound)
		}
	}
}

func PhoneHandlerPost(db *bolt.DB) func(http.ResponseWriter, *http.Request) {
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
