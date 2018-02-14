package main

import (
	"fmt"
	bolt "github.com/coreos/bbolt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

func authed(db *bolt.DB, req *http.Request) bool {
	_, err := SessionGet(db, req, "user")
	return err == nil
}

func RequireAuth(db *bolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		if !authed(db, req) {
			rw.Header().Add("location", "/login")
			http.Error(rw, "You are not logged in.", http.StatusFound)
			return
		}
		handler(rw, req)
	}
}

func RequireNotAuth(db *bolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		if authed(db, req) {
			rw.Header().Add("location", "/")
			http.Error(rw, "You are logged in!", http.StatusFound)
			return
		}
		handler(rw, req)
	}
}

func LoginHandler(db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "POST":
			RequireForm([]string{"username", "password"}, LoginHandlerPost(db))(rw, req)
		default:
			LoginHandlerGet(rw, req)
		}
	}
}

func LoginHandlerPost(db *bolt.DB) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		invalid := func() {
			http.Error(rw, "The username or password you provided was invalid.", http.StatusBadRequest)
		}
		user, err := LoadUser(db, req.Form.Get("username"))
		if err == UserNotFound {
			invalid()
			return
		}
		if err != nil {
			http.Error(rw, "An error occurred validating your credentials.", http.StatusInternalServerError)
			return
		}
		if err := bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(req.Form.Get("password"))); err != nil {
			invalid()
			return
		}
		fmt.Fprintf(rw, "Welcome, %s!", req.Form.Get("username"))
	}
}

func LoginHandlerGet(rw http.ResponseWriter, req *http.Request) {
	const login = `
<!DOCTYPE html>
<html>
  <head>
    <title>vator!</title>
  </head>
  <body>
    Welcome to vator! Vator will motivate you and be not at all creepy. It's a motivator.<br/>
    Perhaps you'd like to log in?<br/>
    <form action="/login" method="POST">
      <input type='text' name='username' placeholder='Username'><br/>
      <input type='password' name='password' placeholder='Password'><br/>
      <input type='submit' value='Log in'>
    </form>
    <br/>
    Or maybe you'd like to <a href='/signup/'>sign up</a>, instead?
  </body>
</html>
`
	rw.Header().Add("content-type", "text/html; charset=utf-8")
	fmt.Fprint(rw, login)
}
