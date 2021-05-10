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
			Page: "graph",
			User: req.Form.Get("user"),
		})
	})
}

type DataPoint struct {
	Samples   []float64
	FiveDay   float64
	ThirtyDay float64
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

		if err == nil && !user.Share {
			var currentUser string
			currentUser, err = models.SessionGet(db, req, "user")
			if err != nil {
				err = fmt.Errorf("user %q is not shared, and request is "+
					"unauthenticated: %w", user.Username, err)
			} else if user.Username != currentUser {
				err = fmt.Errorf("user %q is not shared, and does not belong "+
					"to %q", user.Username, currentUser)
			}
		}

		if err != nil {
			log.Log.Warningf("getting data for user=%q days=%q: %v",
				req.Form.Get("user"), req.Form.Get("days"), err)
			fmt.Fprint(rw, "[]")
			return
		}

		log.Log.Debugf("user %q has %d weights", user.Username, len(user.Weights))

		var start, first time.Time
		if days > 0 {
			start = time.Now().Add(-time.Duration(days) * 24 * time.Hour)
		}

		series := map[time.Time]*DataPoint{}

		// We have a list of time,weight pairs; collect samples by day
		for _, w := range user.Weights {
			// We need 30 days extra to compute moving averages, but anything
			// before that we can disregard.
			if w.Date.Before(start.Add(-30 * 24 * time.Hour)) {
				continue
			}

			t := w.Date.Truncate(24 * time.Hour)

			if first.IsZero() || t.Before(first) {
				first = t
			}

			if dp, ok := series[t]; ok {
				dp.Samples = append(dp.Samples, w.Kgs)
			} else {
				series[t] = &DataPoint{Samples: []float64{w.Kgs}}
			}
		}

		log.Log.Debugf("%d days to compute requested range", len(series))

		if start.IsZero() {
			start = first
		}

		type DataPointExport struct {
			Date      time.Time
			Day       float64
			FiveDay   float64
			ThirtyDay float64
		}

		var ret []DataPointExport
		for t := first; t.Before(time.Now()); t = t.Add(24 * time.Hour) {
			if t.Before(start) {
				continue
			}

			dp, ok := series[t]
			if !ok {
				series[t] = &DataPoint{}
				dp = series[t]
			}
			dp.FiveDay = MovingAverage(t, series, 5)
			dp.ThirtyDay = MovingAverage(t, series, 30)

			ret = append(ret, DataPointExport{
				Date:      t,
				Day:       Mean(dp.Samples),
				FiveDay:   dp.FiveDay,
				ThirtyDay: dp.ThirtyDay,
			})
		}

		log.Log.Debugf("returning %d days", len(ret))

		enc := json.NewEncoder(rw)
		if err := enc.Encode(ret); err != nil {
			log.Log.Warningf("getting data for user=%q days=%q: %v",
				req.Form.Get("user"), req.Form.Get("days"), err)
		}
	})
}

func MovingAverage(t time.Time, data map[time.Time]*DataPoint, days int) float64 {
	var samples []float64
	for d := time.Duration(-days + 1); d <= 0; d++ {
		dp, ok := data[t.Add(d*24*time.Hour)]
		if !ok {
			continue
		}
		if avg := Mean(dp.Samples); avg != 0 {
			samples = append(samples, avg)
		}
	}

	if len(samples) >= days*3/4 {
		return Mean(samples)
	}

	return 0
}

func Mean(in []float64) (out float64) {
	if len(in) == 0 {
		return 0
	}
	for _, f := range in {
		out += f
	}
	return out / float64(len(in))
}
