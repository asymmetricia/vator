package main

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

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
