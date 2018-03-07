package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/bbolt"
	. "github.com/pdbogen/vator/log"
	"net/http"
	"strconv"
	"time"
)

// WithSession returns an http handler function that wraps an underlying handler and amends the request context to
// include a `session` key containing the session ID as a string. A new session is created if the request session is
// missing or invalid.
func WithSession(db *bolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		sid, err := req.Cookie("session")
		if err == nil && sid.Value != "" && SessionExists(db, sid.Value) {
			handler(rw, req.WithContext(context.WithValue(req.Context(), "session", sid.Value)))
			return
		}

		id := make([]byte, 32)
		if _, err := rand.Read(id); err != nil {
			http.Error(rw, "could not generate session ID", http.StatusInternalServerError)
			Log.Errorf("could not get bytes to generate a session ID: %s", err)
			return
		}
		hexid := hex.EncodeToString(id)
		req = req.WithContext(context.WithValue(req.Context(), "session", hexid))
		rw.Header().Add("set-cookie", fmt.Sprintf("session=%s; HttpOnly", hexid))
		SessionSet(db, req, "created", strconv.FormatInt(time.Now().Unix(), 10))
		handler(rw, req)
	}
}

func SessionExists(db *bolt.DB, sid string) bool {
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("sessions"))
		if b == nil {
			return errors.New("no bucket")
		}
		sessionData := b.Get([]byte(sid))
		if sessionData == nil {
			return errors.New("no such session")
		}
		return nil
	})
	return err == nil
}

var KeyDoesNotExist = errors.New("key does not exist")

// SessionGet retrieves the named key from the request's session. If the key does not exist, a blank string and
// err will be KeyDoesNotExist is returned. If some other error occurs, the returned error will be non-nil.
func SessionGet(db *bolt.DB, req *http.Request, key string) (value string, err error) {
	var sid string
	if strsid, ok := req.Context().Value("session").(string); ok {
		sid = strsid
	}
	if sid == "" {
		return "", errors.New("no session ID")
	}
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("sessions"))
		if b == nil {
			return errors.New("no bucket")
		}
		sessionData := b.Get([]byte(sid))
		if sessionData == nil {
			return errors.New("no such session")
		}
		sess := map[string]string{}
		if err := json.Unmarshal(sessionData, &sess); err != nil {
			return fmt.Errorf("corrupt session %q (%q): %s", sid, string(sessionData), err)
		}
		var ok bool
		value, ok = sess[key]
		if !ok {
			return KeyDoesNotExist
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return
}

func SessionDelete(db *bolt.DB, req *http.Request) error {
	var sid string
	if strsid, ok := req.Context().Value("session").(string); ok {
		sid = strsid
	}
	if sid == "" {
		return errors.New("no session ID")
	}
	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return fmt.Errorf("creating sessions bucket: %s", err)
		}
		return b.Delete([]byte(sid))
	})
	return err
}

func SessionSet(db *bolt.DB, req *http.Request, key string, value string) error {
	var sid string
	if strsid, ok := req.Context().Value("session").(string); ok {
		sid = strsid
	}
	if sid == "" {
		return errors.New("no session ID")
	}
	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return fmt.Errorf("creating sessions bucket: %s", err)
		}
		sessionData := b.Get([]byte(sid))
		if sessionData == nil {
			sessionData = []byte("{}")
		}
		sess := map[string]string{}
		if err := json.Unmarshal(sessionData, &sess); err != nil {
			Log.Warningf("corrupt session %q (%q): %s", sid, string(sessionData), err)
		}
		sess[key] = value
		newSessionData, err := json.Marshal(sess)
		if err != nil {
			return fmt.Errorf("rendering JSON: %s", err)
		}
		if err := b.Put([]byte(sid), newSessionData); err != nil {
			return fmt.Errorf("saving session: %s", err)
		}
		return nil
	})
	return err
}
