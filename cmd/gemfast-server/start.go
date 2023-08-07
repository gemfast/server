package cmd

import (
	"time"

	"github.com/gemfast/server/internal/api"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/license"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the gemfast rubygems server",
	Long:  "Reads in the gemfast config file and starts the gemfast rubygems server",
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func start() {
	// Load the config
	cfg := config.NewConfig()

	// Load the license
	license, err := license.NewLicense(cfg)
	check(err)
	log.Info().Msg("starting services")

	// Connect to the database
	database, err := db.NewDB(cfg)
	check(err)
	database.Open()
	defer database.Close()
	database.SaveLicense(license)

	// Start the indexer
	indexer, err := indexer.NewIndexer(cfg, database)
	check(err)
	err = indexer.GenerateIndex()
	check(err)

	// Create the filter
	f := filter.NewFilter(cfg, license)

	// Start the advisory DB updater
	advisoryDB := cve.NewGemAdvisoryDB(cfg, license)
	err = advisoryDB.Refresh()
	check(err)
	ticker := time.NewTicker(24 * time.Hour)
	quit := make(chan struct{})
	go func(advisoryDB *cve.GemAdvisoryDB) {
		log.Info().Msg("starting ruby advisory DB updater")
		for {
			select {
			case <-ticker.C:
				advisoryDB.Refresh()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}(advisoryDB)

	// Start the API
	apiV1Handler := api.NewAPIV1Handler(cfg, database, indexer, f, advisoryDB)
	rubygemsHandler := api.NewRubyGemsHandler(cfg, database, indexer, f, advisoryDB)
	api := api.NewAPI(cfg, database, apiV1Handler, rubygemsHandler)
	api.Run()
}
