package cmd

import (
	"github.com/spf13/cobra"

	"github.com/gemfast/server/internal/api"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/indexer"
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
	// indexer.Get().GenerateIndex()
	err = api.Run()
	if err != nil {
		panic(err)
	}
}
