package main

import (
	"errors"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

func ConsumeState(db *bbolt.DB, state string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(StatesBucket))
		if bucket == nil {
			return errors.New("not found")
		}

		value := bucket.Get([]byte(state))
		if value == nil {
			return errors.New("not found")
		}

		bucket.Delete([]byte(state))

		var expiry time.Time
		if err := expiry.UnmarshalText(value); err != nil {
			return errors.New("corrupt")
		}

		if expiry.Before(time.Now()) {
			return errors.New("expired")
		}

		return nil
	})
}
func SaveState(db *bbolt.DB, state string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(StatesBucket))
		if err != nil {
			return fmt.Errorf("getting `%s` bucket: %s", StatesBucket, err)
		}

		var deletions [][]byte
		err = bucket.ForEach(func(k, v []byte) error {
			var expiry time.Time
			if err := expiry.UnmarshalText(v); err != nil {
				deletions = append(deletions, k)
				return nil
			}
			if expiry.Before(time.Now()) {
				deletions = append(deletions, k)
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
		for _, del := range deletions {
			if err := bucket.Delete(del); err != nil {
				panic(err)
			}
		}

		expiry, err := time.Now().Add(time.Hour).MarshalText()
		if err != nil {
			panic(err)
		}
		return bucket.Put([]byte(state), expiry)
	})
}
