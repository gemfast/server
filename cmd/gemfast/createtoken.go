package cmd

import (
	

	"github.com/spf13/cobra"
)

var createtokenCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API token",
	Run: func(cmd *cobra.Command, args []string) {
		
	},
}

func init() {
	tokenCmd.AddCommand(createtokenCmd)
}