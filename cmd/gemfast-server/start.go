package cmd

import (
	"context"
	"os"
	"time"

	"github.com/gemfast/server/internal/api"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/telemetry"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// serviceVersion is overridable at link time (-ldflags "-X .../cmd.serviceVersion=v1.2.3").
var serviceVersion = "dev"

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the gemfast rubygems server",
	Long:  "Reads in the gemfast config file and starts the gemfast rubygems server",
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}

var configPath string

func init() {
	startCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to the config file")
	// Expose the same flag on the root command so it works when invoking
	// gemfast-server with no subcommand (defaults to start).
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to the config file")
	rootCmd.AddCommand(startCmd)
}

func start() {
	if configPath != "" {
		os.Setenv("GEMFAST_CONFIG_FILE", configPath)
	}
	// Load the config
	cfg := config.NewConfig()
	log.Info().Msg("starting services")

	// Initialize OpenTelemetry tracing (exports to Honeycomb by default).
	ctx := context.Background()
	shutdownTracer, err := telemetry.Init(ctx, serviceVersion, cfg)
	if err != nil {
		log.Warn().Err(err).Msg("failed to initialize tracing; continuing without it")
	} else {
		defer func() {
			if err := shutdownTracer(context.Background()); err != nil {
				log.Warn().Err(err).Msg("error shutting down tracer provider")
			}
		}()
	}

	// Connect to the database
	database, err := db.NewDB(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	database.Open()
	defer database.Close()

	// Start the indexer
	indexer, err := indexer.NewIndexer(cfg, database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create indexer")
	}
	err = indexer.GenerateIndex()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to generate index")
	}

	// Create the filter
	f := filter.NewFilter(cfg)

	// Start the advisory DB updater
	advisoryDB := cve.NewGemAdvisoryDB(cfg)
	err = advisoryDB.Refresh()
	if err != nil {
		log.Warn().Err(err).Msg("failed to refresh advisory DB")
	}
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
