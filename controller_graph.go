package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"go.etcd.io/bbolt"
)

func Graph(db *bbolt.DB) func(rw http.ResponseWriter, req *http.Request) {
	return RequireForm([]string{"user"}, func(rw http.ResponseWriter, req *http.Request) {
		TemplateGet(rw, req, "graph.tmpl", TemplateContext{
			User: req.Form.Get("user"),
		})
	})
}

func Data(db *bbolt.DB) func(rw http.ResponseWriter, req *http.Request) {
	return RequireForm([]string{"user"}, func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Add("content-type", "application/json")
		days := 365
		var err error
		if dayStr := req.Form.Get("days"); dayStr != "" {
			days, err = strconv.Atoi(dayStr)
		}

		var user *models.User
		if err == nil {
			user, err = models.LoadUser(db, req.Form.Get("user"))
		}

		if err != nil {
			log.Log.Warningf("getting data for user=%q days=%q: %v",
				req.Form.Get("user"), req.Form.Get("days"), err)
			fmt.Fprint(rw, "[]")
			return
		}

		log.Log.Debugf("user %q has %d weights", user.Username, len(user.Weights))

		var start time.Time
		if days > 0 {
			start = time.Now().Add(-time.Duration(days) * 24 * time.Hour)
		}

		ret := []models.Weight{}
		for _, w := range user.Weights {
			if w.Date.Before(start) {
				continue
			}
			ret = append(ret, w)
		}

		enc := json.NewEncoder(rw)
		if err := enc.Encode(ret); err != nil {
			log.Log.Warningf("getting data for user=%q days=%q: %v",
				req.Form.Get("user"), req.Form.Get("days"), err)
		}
	})
}
