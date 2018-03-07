package main

import (
	"flag"
	"github.com/coreos/bbolt"
	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"os"
	"sort"
	"time"
)

var log = Log

func main() {
	skiplast := flag.Bool("skip-last", false, "if true, skip updating lastweight")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatalf("usage: %s <username>", os.Args[0])
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

	if len(user.Weights) >= 2 {

		sort.Sort(models.WeightsByDate(user.Weights))
		log.Infof("dropping weight %v", user.Weights[len(user.Weights)-1])
		user.Weights = user.Weights[0 : len(user.Weights)-1]
		if !*skiplast {
			user.LastWeight = user.Weights[len(user.Weights)-1].Date.Add(time.Minute)
			log.Infof("set lastweight to %v", user.LastWeight)
		}
	} else {
		log.Info("clearing last couple weights")
		user.Weights = []models.Weight{}
	}

	if err := user.Save(db); err != nil {
		log.Fatalf("saving user: %s", err)
	}
	log.Info("done!")
}
