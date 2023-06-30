package cmd

import (
	"time"

	"github.com/gemfast/server/internal/api"
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
	l, err := license.NewLicense()
	check(err)
	log.Info().Msg("starting services")
	err = db.Connect(l)
	check(err)
	defer db.BoltDB.Close()
	check(err)
	err = indexer.InitIndexer()
	check(err)
	err = indexer.Get().GenerateIndex()
	check(err)
	err = filter.InitFilter(l)
	check(err)
	cve.InitRubyAdvisoryDB(l)
	ticker := time.NewTicker(24 * time.Hour)
	quit := make(chan struct{})
	go func() {
		log.Info().Msg("starting ruby advisory DB updater")
		for {
			select {
			case <-ticker.C:
				cve.InitRubyAdvisoryDB(l)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	if !l.Validated {
		log.Warn().Msg("no valid license found, starting in trial mode")
	}
	err = api.Run(l)
	check(err)
}
