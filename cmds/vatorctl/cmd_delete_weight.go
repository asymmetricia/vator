package main

import (
	"strconv"

	"github.com/asymmetricia/vator/log"
	"github.com/asymmetricia/vator/models"
	"github.com/spf13/cobra"
)

var cmdDeleteWeight = &cobra.Command{
	Use:  "delete-weight username weight-timestamp",
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		user, err := models.LoadUser(Db(), args[0])
		if err != nil {
			log.Log.Fatalf("loading user %q: %v", args[0], err)
		}

		target, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			log.Log.Fatalf("expected unix timestamp for target timestamp "+
				"but could not parse %q: %v", args[1], err)
		}

		var deleted = true
		for deleted {
			deleted = false
			for i, weight := range user.Weights {
				if weight.Date.Unix() == target {
					log.Log.Infof("deleting weight %+v", weight)
					user.Weights = append(user.Weights[:i], user.Weights[i+1:]...)
					deleted = true
					break
				}
			}
		}

		if err := user.Save(Db()); err != nil {
			log.Log.Fatalf("could not save user after modifying weights list: %v", err)
		}

		return
	},
}

func init() {
	root.AddCommand(cmdDeleteWeight)
}
