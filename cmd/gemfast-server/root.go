package cmd

import (
	"github.com/spf13/cobra"

	"github.com/gemfast/server/internal/api"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	// "github.com/gemfast/server/internal/license"
	"github.com/rs/zerolog/log"
)

var rootCmd = &cobra.Command{
	Use:   "gemfast",
	Short: "gemfast is a private rubygems server",
}

func init() {
	config.InitConfig()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func Execute() {
	log.Info().Msg("starting services")
	err := db.Connect()
	check(err)
	defer db.BoltDB.Close()
	err = indexer.InitIndexer()
	check(err)
	err = indexer.Get().GenerateIndex()
	check(err)
	err = filter.InitFilter()
	check(err)
	// err = license.ValidateLicenseKey()
	// check(err)
	err = api.Run()
	check(err)
}
