package main

import (
	"time"

	"github.com/asymmetricia/vator/log"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

var root = &cobra.Command{
	Use: "vatorctl",
}

var VatorctlConfig struct {
	BoltDbPath string
}

var db *bbolt.DB

func Db() *bbolt.DB {
	if db != nil {
		return db
	}

	openedDb, err := bbolt.Open(VatorctlConfig.BoltDbPath, 0600, &bbolt.Options{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Log.Fatalf("could not open db at %q: %v", VatorctlConfig.BoltDbPath, err)
	}
	db = openedDb
	return openedDb
}

func main() {
	root.PersistentFlags().StringVar(
		&VatorctlConfig.BoltDbPath,
		"db-path",
		"/opt/vator/vator.db",
		"path to the vator.db file to operate on",
	)

	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	if err := root.Execute(); err != nil {
		log.Log.Fatal(err)
	}
}
