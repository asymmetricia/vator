package main

import (
	"flag"
	"github.com/coreos/bbolt"
	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"os"
)

var log = Log

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatalf("usage: %s <password>", os.Args[0])
	}

	db, err := bolt.Open("vator.db", 0600, nil)
	if err != nil {
		Log.Fatalf("opening bolt db file vator.db: %s", err)
	}
	defer db.Close()
	user, err := models.LoadUser(db, "pdbogen")
	if err != nil {
		log.Fatalf("loading pdbogen: %s", err)
	}

	user.SetPassword(flag.Arg(0))

	if err := user.Save(db); err != nil {
		log.Fatalf("saving user: %s", err)
	}
	log.Info("done!")
}
