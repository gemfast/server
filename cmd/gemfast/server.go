package cmd

import (
	"github.com/spf13/cobra"

	"github.com/gscho/gemfast/internal/api"
	"github.com/gscho/gemfast/internal/db"
	"github.com/gscho/gemfast/internal/indexer"

	"github.com/rs/zerolog/log"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the gemfast server.",
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func serve() {
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
