package main

import (
	"flag"
	"github.com/coreos/bbolt"
	. "github.com/pdbogen/vator/log"
	"github.com/pdbogen/vator/models"
	"os"
	"sort"
	"strconv"
)

var log = Log

func main() {
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

	sort.Sort(models.WeightsByDate(user.Weights))

	for i := 0; i < flag.NArg(); i++ {
		kg, err := strconv.ParseFloat(flag.Arg(i), 64)
		if err != nil {
			log.Fatalf("%q isn't a number!", flag.Arg(i))
		}
		user.Weights[len(user.Weights)-1-i].Kgs = kg
	}

	if err := user.Save(db); err != nil {
		log.Fatalf("saving user: %s", err)
	}

	log.Info("done!")
}
