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
	username := flag.String("username", "", "required; which user will be modified")
	password := flag.String("password", "", "if non-blank, password will be set as given")
	refreshToken := flag.String("refresh-token", "", "if non-blank, refresh token will be set as given")
	flag.Parse()

	if *username == "" {
		log.Error("username was blank, but is required")
		flag.Usage()
		os.Exit(1)
	}

	if *password == "" && *refreshToken == "" {
		log.Error("-password and/or -refresh-token are required, but both are blank")
		flag.Usage()
		os.Exit(1)
	}

	db, err := bbolt.Open("vator.db", 0600, nil)
	if err != nil {
		Log.Fatalf("opening bolt db file vator.db: %s", err)
	}
	defer db.Close()
	user, err := models.LoadUser(db, *username)
	if err != nil {
		log.Fatalf("loading %q: %s", *username, err)
	}

	if *password != "" {
		if err := user.SetPassword(*password); err != nil {
			log.Fatal(err)
		}
	}
	if *refreshToken != "" {
		user.RefreshSecret = *refreshToken
	}

	if err := user.Save(db); err != nil {
		log.Fatalf("saving user: %s", err)
	}
	log.Info("done!")
}
