package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gemfast/server/internal/config"
	"github.com/rs/zerolog/log"
)

var rootCmd = &cobra.Command{
	Use:   "gemfast-server",
	Short: "Gemfast is a rubygems server written in Go",
}

func init() {
	config.LoadConfig()
}

func check(err error) {
	if err != nil {
		log.Error().Err(err).Msg("error starting gemfast server")
		os.Exit(1)
	}
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	err := rootCmd.Execute()
	check(err)
}
