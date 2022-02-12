package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	. "github.com/asymmetricia/vator/log"
	"go.etcd.io/bbolt"
)

// WithNewSession returns an http handler function that wraps an underlying
// handler and amends the request context to include a `session` key
// containing the session ID as a string. A new session is always created,
// copying from an existing session, if any. The old session is deleted.
func WithNewSession(db *bbolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		id := make([]byte, 32)
		if _, err := rand.Read(id); err != nil {
			http.Error(rw, "could not generate session ID", http.StatusInternalServerError)
			Log.Errorf("could not get bytes to generate a session ID: %s", err)
			return
		}
		hexid := hex.EncodeToString(id)
		req = req.WithContext(context.WithValue(req.Context(), "session", hexid))
		rw.Header().Add("set-cookie", fmt.Sprintf("session=%s; HttpOnly", hexid))

		sid, err := req.Cookie("session")
		if err == nil && sid.Value != "" && SessionExists(db, sid.Value) {
			SessionCopy(db, sid.Value, hexid)
			SessionDelete(db, sid.Value)
		}

		SessionSet(db, req, "created", strconv.FormatInt(time.Now().Unix(), 10))

		handler(rw, req)
	}
}

func SessionCopy(db *bbolt.DB, old, new string) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return fmt.Errorf("creating sessions bucket: %s", err)
		}
		sessionData := b.Get([]byte(old))
		sess := map[string]string{}
		if sessionData != nil {
			if err := json.Unmarshal(sessionData, &sess); err != nil {
				Log.Warningf("corrupt session %q (%q): %s", old, string(sessionData), err)
			}
		}

		newSessionData, err := json.Marshal(sess)
		if err != nil {
			return fmt.Errorf("rendering JSON: %s", err)
		}
		if err := b.Put([]byte(new), newSessionData); err != nil {
			return fmt.Errorf("saving session: %s", err)
		}
		return nil
	})
	return err
}

// WithSession returns an http handler function that wraps an underlying handler and amends the request context to
// include a `session` key containing the session ID as a string. A new session is created if the request session is
// missing or invalid.
func WithSession(db *bbolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
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

func SessionExists(db *bbolt.DB, sid string) bool {
	err := db.View(func(tx *bbolt.Tx) error {
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
func SessionGet(db *bbolt.DB, req *http.Request, key string) (value string, err error) {
	v, e := SessionGetMulti(db, req, []string{key})
	return v[0], e
}

// SessionGetMulti retrieves the named keys from the request's session. The returned slice will contain one entry for
// each requested key, but keys that do not exist will be blank. If any keys do not exist or another error occurs,
// values will still be populated with a number of entries equal to len(keys), but err will be non-nil.
func SessionGetMulti(db *bbolt.DB, req *http.Request, keys []string) (values []string, err error) {
	values = make([]string, len(keys))
	var sid string
	if strsid, ok := req.Context().Value("session").(string); ok {
		sid = strsid
	}
	if sid == "" {
		return values, errors.New("no session ID")
	}
	err = db.View(func(tx *bbolt.Tx) error {
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
		var err error
		for i, k := range keys {
			value, ok := sess[k]
			if !ok {
				err = KeyDoesNotExist
			}
			values[i] = value
		}
		return err
	})
	return values, err
}

func SessionDeleteReq(db *bbolt.DB, req *http.Request) error {
	var sid string
	if strsid, ok := req.Context().Value("session").(string); ok {
		sid = strsid
	}
	if sid == "" {
		return errors.New("no session ID")
	}
	return SessionDelete(db, sid)
}

func SessionDelete(db *bbolt.DB, sid string) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return fmt.Errorf("creating sessions bucket: %s", err)
		}
		return b.Delete([]byte(sid))
	})
	return err
}

func SessionSet(db *bbolt.DB, req *http.Request, key string, value string) error {
	return SessionSetMulti(db, req, []string{key}, []string{value})
}

func SessionSetMulti(db *bbolt.DB, req *http.Request, keys []string, values []string) error {
	if len(keys) != len(values) {
		return fmt.Errorf("length mismatch: len(keys) %d != len(values) %d", len(keys), len(values))
	}

	var sid string
	if strsid, ok := req.Context().Value("session").(string); ok {
		sid = strsid
	}
	if sid == "" {
		return errors.New("no session ID")
	}
	err := db.Update(func(tx *bbolt.Tx) error {
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
		for i, k := range keys {
			sess[k] = values[i]
		}
		newSessionData, err := json.Marshal(sess)
		if err != nil {
			return fmt.Errorf("rendering JSON: %s", err)
		}
		if err := b.Put([]byte(sid), newSessionData); err != nil {
			return fmt.Errorf("saving session: %s", err)
		}
		log.Debugf("saved session %q -> %q", sid, string(sessionData))
		return nil
	})
	return err
}
