package main

import (
	"encoding/json"
	"errors"
	bolt "github.com/coreos/bbolt"
)

type User struct {
	Username       string
	HashedPassword []byte
}

func (u *User) Save(db *bolt.DB) error {
	return errors.New("unimplemented")
}

var UserNotFound error = errors.New("user not found")

func LoadUser(db *bolt.DB, username string) (*User, error) {
	var user *User
	err := db.View(func(tx *bolt.Tx) error {
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
			log.Errorf("user record for %q (%q) corrupt: %s", username, string(u), err)
			return UserNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}
