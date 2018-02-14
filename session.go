package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	bolt "github.com/coreos/bbolt"
	"net/http"
	"strconv"
	"time"
)

// WithSession amends the request context to include a `session` key containing the session ID as a string. A new
// session is created if the request session is mising or invalid.
func WithSession(db *bolt.DB, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		sid, err := req.Cookie("session")
		if err == nil && SessionExists(db, sid.Value) {
			handler(rw, req.WithContext(context.WithValue(req.Context(), "session", sid)))
			return
		}

		id := make([]byte, 32)
		if _, err := rand.Read(id); err != nil {
			http.Error(rw, "could not generate session ID", http.StatusInternalServerError)
			log.Errorf("could not get bytes to generate a session ID: %s", err)
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
		session_data := b.Get([]byte(sid))
		if session_data == nil {
			return errors.New("no such session")
		}
		return nil
	})
	return err == nil
}

// SessionGet retrieves the named key from the request's session. If the key does not exist or some other error occurs,
// the returned error will be non-nil.
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
		session_data := b.Get([]byte(sid))
		if session_data == nil {
			return errors.New("no such session")
		}
		sess := map[string]string{}
		if err := json.Unmarshal(session_data, &sess); err != nil {
			return fmt.Errorf("corrupt session %q (%q): %s", sid, string(session_data), err)
		}
		value = sess[key]
		return nil
	})
	if err != nil {
		return "", err
	}
	return
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
		session_data := b.Get([]byte(sid))
		if session_data == nil {
			session_data = []byte("{}")
		}
		sess := map[string]string{}
		if err := json.Unmarshal(session_data, sess); err != nil {
			log.Warningf("corrupt session %q (%q): %s", sid, string(session_data), err)
		}
		sess[key] = value
		new_session_data, err := json.Marshal(sess)
		if err != nil {
			return fmt.Errorf("rendering JSON: %s", err)
		}
		if err := b.Put([]byte(sid), new_session_data); err != nil {
			return fmt.Errorf("saving session: %s", err)
		}
		return nil
	})
	return err
}
