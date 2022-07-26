package main

import (
	"fmt"

	"github.com/asymmetricia/vator/models"
	"github.com/spf13/cobra"
)

var listUsers = &cobra.Command{
	Use:   "list-users",
	Short: "list users in the DB",
	Run: func(cmd *cobra.Command, args []string) {
		for _, user := range models.GetUsers(Db()) {
			fmt.Println(user.Username)
		}
	},
}

func init() {
	root.AddCommand(listUsers)
}
