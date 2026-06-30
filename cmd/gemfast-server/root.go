package cmd

import (
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gemfast-server",
	Short: "Gemfast is a rubygems server written in Go",
	// When invoked with no subcommand, default to running the server (same as "start").
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to execute command")
	}
}
