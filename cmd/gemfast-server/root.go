package cmd

import (
	"github.com/spf13/cobra"

	"github.com/rs/zerolog/log"
)

var rootCmd = &cobra.Command{
	Use:   "gemfast-server",
	Short: "Gemfast is a rubygems server written in Go",
}

func check(err error) {
	if err != nil {
		log.Fatal().Err(err).Msg("error starting gemfast server")
	}
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	err := rootCmd.Execute()
	check(err)
}
