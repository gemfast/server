package cmd

import (
	

	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "api-token",
	Short: "Commands for working with api tokens",
}

func init() {
	rootCmd.AddCommand(tokenCmd)
}