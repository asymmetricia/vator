package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/cbroglie/mustache"
	"github.com/jrmycanady/nokiahealth"
	. "github.com/pdbogen/vator/log"
	errors2 "github.com/pkg/errors"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

const PoundsFromKg = 2.2046226218

func init() {
	rand.Seed(time.Now().UnixNano())
}

type User struct {
	Username       string
	HashedPassword []byte
	LastWeight     time.Time
	Weights        []Weight
	Phone          string

	OauthTime     time.Time
	AccessToken   string
	RefreshSecret string
	TokenExpiry   time.Time

	Kgs          bool
	LastSummary  time.Time
	TimezoneName string
}

type Weight struct {
	Date time.Time
	Kgs  float64
}

var UserNotFound = errors.New("user not found")

func (u *User) NokiaUser(db *bbolt.DB, client *nokiahealth.Client) (*nokiahealth.User, error) {
	if u.RefreshSecret == "" {
		return nil, errors.New("not linked")
	}

	return client.NewUserFromRefreshToken(context.Background(), u.RefreshSecret), nil
}

func LoadUserRequest(db *bbolt.DB, req *http.Request) (*User, error) {
	if user, ok := req.Context().Value("user").(string); ok {
		return LoadUser(db, user)
	}
	return nil, errors.New("no user in request context")
}

func LoadUser(db *bbolt.DB, username string) (*User, error) {
	var user *User
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			return UserNotFound
		}
		u := b.Get([]byte(username))
		if u == nil {
			return UserNotFound
		}
		user = &User{}
		if err := json.Unmarshal(u, user); err != nil {
			Log.Errorf("user record for %q (%q) corrupt: %s", username, string(u), err)
			return UserNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u *User) Save(db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return fmt.Errorf("opening users bucket: %s", err)
		}
		user, err := json.Marshal(u)
		if err != nil {
			return fmt.Errorf("marshalling user into JSON: %s", err)
		}
		return b.Put([]byte(u.Username), user)
	})
}

func (u *User) SetPassword(newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 11)
	if err != nil {
		return fmt.Errorf("hashing password: %s", err)
	}
	u.HashedPassword = hash
	return nil
}

func GetUsers(db *bbolt.DB) []*User {
	var users []*User
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			return nil
		}
		err := b.ForEach(func(k, v []byte) error {
			u := &User{}
			if err := json.Unmarshal(v, u); err != nil {
				Log.Warning("skipping unparseable user %s: %q", string(k), string(v))
				return nil
			}
			if u.RefreshSecret == "" {
				Log.Debugf("skipping unlinked user %q", string(k))
				return nil
			}
			users = append(users, u)
			return nil
		})
		if err != nil {
			return fmt.Errorf("unexpected error iterating over contents of users bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		Log.Errorf("unexpected, but error getting list of users: %s", err)
	}
	return users
}

func (u *User) SaveRefreshToken(db *bbolt.DB, nuser *nokiahealth.User) {
	if nuser.OauthToken.RefreshToken == u.RefreshSecret {
		log.Debugf("user %q refresh token unchanged", u.Username)
		return
	}

	log.Debugf("saving updated refresh secret for user %q", u.Username)
	u.RefreshSecret = nuser.OauthToken.RefreshToken
	if err := u.Save(db); err != nil {
		log.Errorf("saving user due to refresh token update: %v", err)
	}
}

func (u *User) GetWeights(db *bbolt.DB, withings *nokiahealth.Client) ([]nokiahealth.Weight, error) {
	return u.GetWeightsSince(db, withings, time.Now().AddDate(0, 0, -37))
}

func (u *User) GetWeightsSince(db *bbolt.DB, withings *nokiahealth.Client, since time.Time) ([]nokiahealth.Weight,
	error) {
	nuser, err := u.NokiaUser(db, withings)
	if err != nil {
		return nil, err
	}

	measureResp, err := nuser.GetBodyMeasures(&nokiahealth.BodyMeasuresQueryParams{StartDate: &since})
	u.SaveRefreshToken(db, nuser)
	if err != nil {
		return nil, err
	}
	measures := measureResp.ParseData()

	return measures.Weights, nil
}

// MovingAverageWeight calculates the moving average of the user's weight. `days` specifies the size of the window, and
// `shift` specifies how many days in the past the window should be moved. An error will be returned if there are not
// enough samples.
func (u User) MovingAverageWeight(days int, shift int) (float64, error) {
	var samples []float64
	for cursor := 0; cursor < days; cursor++ {
		var daySamples []float64
		targetDay := time.Now().AddDate(0, 0, -shift-cursor).Truncate(24 * time.Hour)
		for i := len(u.Weights) - 1; i >= 0; i-- {
			weightDay := u.Weights[i].Date.Truncate(24 * time.Hour)
			if !targetDay.Equal(weightDay) {
				continue
			}
			daySamples = append(daySamples, u.Weights[i].Kgs)
		}
		var sum float64
		for _, s := range daySamples {
			sum += s
		}
		if len(daySamples) > 0 {
			samples = append(samples, sum/float64(len(daySamples)))
		}
	}
	if len(samples) >= days*6/10 {
		sum := float64(0)
		for _, s := range samples {
			sum += s
		}
		return sum / float64(len(samples)), nil
	}
	return 0, errors.New("insufficient samples")
}

func (u User) sendSms(twilio *Twilio, message string) error {
	if u.Phone == "" {
		return fmt.Errorf("user %q has no registered phone, cannot toast", u.Username)
	}

	if twilio == nil {
		return errors.New("twilio mis- or unconfigured, cannot toast")
	}

	if err := twilio.SendSms(u.Phone, message); err != nil {
		return errors2.WithMessagef(err, "sending toast to %s at %s", u.Username, u.Phone)
	}

	return nil
}

var InsufficientData = errors.New("insufficient data")
var Unwarranted = errors.New("unwarranted")

func (u User) toastN(days int, twilio *Twilio, encourage bool) error {
	current, err := u.MovingAverageWeight(days, 0)
	if err != nil {
		return InsufficientData
	}
	prev, err := u.MovingAverageWeight(days, 1)
	if err != nil {
		return InsufficientData
	}

	Log.Infof("%d-day, previous: %.2f, now: %.2f", days, prev, current)

	englishDay := strconv.Itoa(days)
	switch days {
	case 5:
		englishDay = "five"
	case 30:
		englishDay = "thirty"
	}

	ctx := map[string]string{
		"days":      englishDay,
		"direction": "down",
		"delta":     u.FormatKg(prev - current),
		"final":     u.FormatKg(current),
		"unit":      u.Unit(),
	}

	var tmpl string

	if prev <= current {
		if !encourage {
			return Unwarranted
		}

		log.Infof("sending %d-day encouragement to %s", days, u.Username)
		ctx["direction"] = "up"
		ctx["delta"] = u.FormatKg(current - prev)
		tmpl = encourageToasts[rand.Intn(len(encourageToasts))]
	} else {
		log.Infof("sending %d-day toast for %s!", days, u.Username)
		tmpl = toasts[rand.Intn(len(toasts))]
	}

	msg, err := mustache.Render(tmpl, ctx)
	if err != nil {
		log.Errorf("rendering toast template %q: %s", tmpl, err)
		return errors.New("template failed")
	}
	if err := u.sendSms(twilio, msg); err != nil {
		log.Errorf("failed sending toast: %s", err)
	}

	return nil
}

func (u User) Toast(twilio *Twilio) {
	if len(u.Weights) == 0 {
		log.Info("no weights logged for %s, cannot toast", u.Username)
		return
	}

	sort.Slice(u.Weights, func(i, j int) bool { return u.Weights[i].Date.Before(u.Weights[j].Date) })

	fiveErr := u.toastN(5, twilio, false)
	if fiveErr == nil {
		return
	}

	thirtyErr := u.toastN(30, twilio, true)
	if thirtyErr == nil {
		return
	}

	if fiveErr == InsufficientData && thirtyErr == InsufficientData {
		log.Debugf("encouraging %q to provide more data", u.Username)
		// send not enough data message
		msg := notEnoughData[rand.Intn(len(notEnoughData))]
		if err := u.sendSms(twilio, msg); err != nil {
			log.Errorf("failed sending toast: %w", err)
		}
		return
	}
	log.Debugf("confusing toast results for %q: 5=%q, 30=%q", u.Username, fiveErr, thirtyErr)
}

func (u *User) Summary(twilio *Twilio, db *bbolt.DB, force bool) {
	userTz := u.Timezone()
	// Weekly summaries only on Sunday
	if !force && time.Now().In(userTz).Weekday() != time.Sunday {
		return
	}

	// One summary per day
	if !force && time.Now().Sub(u.LastSummary).Hours() < 25 {
		return
	}

	log.Debugf(
		"summary: today is %s and last summary was %.01f hours ago; producing summary for %q",
		time.Now().In(userTz).Weekday().String(),
		time.Now().Sub(u.LastSummary).Hours(),
		u.Username,
	)

	msg := fmt.Sprintf("Since %s:",
		time.Now().In(userTz).AddDate(0, 0, -7).Format("Mon Jan 2 2006"))

	for _, delta := range []int{5, 30} {
		msg += fmt.Sprintf("\n%d-day Average: ", delta)
		now, err := u.MovingAverageWeight(delta, 0)
		if err != nil {
			log.Errorf("calculating current %d-day moving average for %q: %w", delta, u.Username, err)
			msg += "insufficient data :("
			continue
		}

		then, err := u.MovingAverageWeight(delta, 7)
		if err != nil {
			log.Errorf("calculating 7-day-shifted %d-day moving average for %q: %w", delta, u.Username, err)
			msg += "insufficient data :("
			continue
		}

		if now <= then {
			msg += " down "
		} else {
			msg += " up "
		}

		msg += u.FormatKg(math.Abs(now-then)) + u.Unit()
	}

	weighs := 0
	for _, w := range u.Weights {
		if w.Date.After(time.Now().Add(-7 * 24 * time.Hour)) {
			weighs++
		}
	}
	msg += fmt.Sprintf("\n%d weigh-ins on record", weighs)

	if err := u.sendSms(twilio, msg); err != nil {
		log.Errorf("failed sending weekly summary: %w", err)
		return
	}

	u.LastSummary = time.Now()
	if err := u.Save(db); err != nil {
		log.Errorf("failed to update LastSummary date: %w", err)
	}
}

func (u User) FormatKg(kgs float64) string {
	if u.Kgs {
		return fmt.Sprintf("%0.1f", kgs)
	} else {
		return fmt.Sprintf("%0.1f", PoundsFromKg*kgs)
	}
}

func (u User) Unit() string {
	if u.Kgs {
		return "kg"
	}
	return "lb"
}

func (u User) Timezone() *time.Location {
	if u.TimezoneName == "" {
		u.TimezoneName = "America/Los_Angeles"
	}

	loc, err := time.LoadLocation(u.TimezoneName)
	if err != nil {
		log.Error("user %q has bad time zone %q: %w", u.TimezoneName, err)
		loc, err = time.LoadLocation("America/Los_Angeles")
	}

	if err != nil {
		log.Error("could not load tz America/Los_Angeles: %w", err)
		return time.UTC
	}

	return loc
}
