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

func Execute() {
	log.Info().Msg("starting services")
	err := db.Connect()
	if err != nil {
		panic(err)
	}
	defer db.BoltDB.Close()
	err = indexer.InitIndexer()
	if err != nil {
		panic(err)
	}
	err = indexer.Get().GenerateIndex()
	if err != nil {
		panic(err)
	}
	filter.InitFilter()
	// err = license.ValidateLicenseKey()
	// if err != nil {
	// 	panic(err)
	// }
	err = api.Run()
	if err != nil {
		panic(err)
	}
}
