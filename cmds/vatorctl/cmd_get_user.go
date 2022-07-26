package main

import (
	"fmt"

	"github.com/asymmetricia/vator/log"
	"github.com/asymmetricia/vator/models"
	"github.com/spf13/cobra"
)

var getUserConfig struct {
	ListWeights bool
}

var getUser = &cobra.Command{
	Use:   "get-user username",
	Short: "retrieve information about the given user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		user, err := models.LoadUser(Db(), args[0])
		if err != nil {
			log.Log.Fatalf("loading user %q: %v", args[0], err)
		}

		fmt.Println("username: ", user.Username)
		fmt.Println("weights:  ", len(user.Weights))

		if getUserConfig.ListWeights {
			for _, weight := range user.Weights {
				fmt.Println("  - ", weight.Date.Unix(), " ", weight.Kgs, "kg")
			}
		}
	},
}

func init() {
	getUser.Flags().BoolVar(
		&getUserConfig.ListWeights,
		"list-weights",
		false,
		"if true, list all weights for user",
	)
	root.AddCommand(getUser)
}
