package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cbroglie/mustache"
	"github.com/coreos/bbolt"
	"github.com/jrmycanady/nokiahealth"
	. "github.com/pdbogen/vator/log"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"
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
	AccessToken    string
	RefreshSecret  string
	TokenExpiry    time.Time
	Kgs            bool
}

type Weight struct {
	Date time.Time
	Kgs  float64
}

var UserNotFound = errors.New("user not found")

func (u *User) NokiaUser(client nokiahealth.Client) (*nokiahealth.User, error) {
	if u.RefreshSecret == "" {
		return nil, errors.New("not linked")
	}
	return client.NewUserFromRefreshToken(context.Background(), u.AccessToken, u.RefreshSecret)
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

func GetUsers(db *bbolt.DB) []User {
	users := []User{}
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			return nil
		}
		err := b.ForEach(func(k, v []byte) error {
			var u User
			if err := json.Unmarshal(v, &u); err != nil {
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
			return fmt.Errorf("unexpected error iterating over contents of users bicket: %s", err)
		}
		return nil
	})
	if err != nil {
		Log.Errorf("unexpected, but error getting list of users: %s", err)
	}
	return users
}

func (u *User) GetWeights(nokia nokiahealth.Client) ([]nokiahealth.Weight, error) {
	return u.GetWeightsSince(nokia, time.Now().AddDate(0, 0, -200))
}

func (u *User) GetWeightsSince(nokia nokiahealth.Client, since time.Time) ([]nokiahealth.Weight, error) {
	nuser, err := u.NokiaUser(nokia)
	if err != nil {
		return nil, err
	}

	measureResp, err := nuser.GetBodyMeasures(&nokiahealth.BodyMeasuresQueryParams{StartDate: &since})
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
	samples := []float64{}
	for mod := 0; mod < days; mod++ {
		daySamples := []float64{}
		for i := len(u.Weights) - 1; i >= 0; i-- {
			tgt := time.Now().AddDate(0, 0, -shift-mod).Truncate(24 * time.Hour)
			wDate := u.Weights[i].Date.Truncate(24 * time.Hour)
			if tgt.Equal(wDate) {
				daySamples = append(daySamples, u.Weights[i].Kgs)
			}
			if tgt.After(wDate) {
				break
			}
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

func (u User) sendSms(twilio *Twilio, message string) {
	if u.Phone == "" {
		Log.Warningf("user %q has no registered phone, cannot toast", u.Username)
		return
	}

	if twilio == nil {
		log.Warning("twilio mis- or unconfigured, cannot toast")
		return
	}

	if err := twilio.SendSms(u.Phone, message); err != nil {
		log.Errorf("sending toast to %s at %s: %s", u.Username, u.Phone, err)
	}
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
	u.sendSms(twilio, msg)

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
		// send not enough data message
		msg := notEnoughData[rand.Intn(len(notEnoughData))]
		u.sendSms(twilio, msg)
		return
	}
}

func (u User) FormatKg(kgs float64) string {
	if u.Kgs {
		return fmt.Sprintf("%0.1fkg", kgs)
	} else {
		return fmt.Sprintf("%0.1flb", PoundsFromKg*kgs)
	}
}
