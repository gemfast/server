package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gemfast/server/internal/api"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/license"
	"github.com/rs/zerolog/log"
)

var rootCmd = &cobra.Command{
	Use:   "gemfast",
	Short: "gemfast is a private rubygems server",
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
	err := license.ValidateLicenseKey()
	check(err)
	log.Info().Msg("starting services")
	cve.InitRubyAdvisoryDB()
	err = db.Connect()
	check(err)
	defer db.BoltDB.Close()
	err = indexer.InitIndexer()
	check(err)
	err = indexer.Get().GenerateIndex()
	check(err)
	err = filter.InitFilter()
	check(err)
	err = api.Run()
	check(err)
}
