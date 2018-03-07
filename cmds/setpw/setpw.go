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

	if flag.NArg() != 2 {
		log.Fatalf("usage: %s <username> <password>", os.Args[0])
	}

	db, err := bolt.Open("vator.db", 0600, nil)
	if err != nil {
		Log.Fatalf("opening bolt db file vator.db: %s", err)
	}
	defer db.Close()
	user, err := models.LoadUser(db, flag.Arg(0))
	if err != nil {
		log.Fatalf("loading %q: %s", flag.Arg(0), err)
	}

	user.SetPassword(flag.Arg(1))

	if err := user.Save(db); err != nil {
		log.Fatalf("saving user: %s", err)
	}
	log.Info("done!")
}
