package main

import (
	"fmt"

	"github.com/gscho/gemfast/internal/api"
	"github.com/gscho/gemfast/internal/db"
	"github.com/gscho/gemfast/internal/indexer"
	"github.com/gscho/gemfast/internal/models"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("GEMFAST")
	viper.SetDefault("dir", "/var/gemfast")
	viper.SetDefault("gem_dir", fmt.Sprintf("%s/gems", viper.Get("dir")))
	viper.SetDefault("db_dir", fmt.Sprintf("db"))
	viper.AutomaticEnv()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	err := db.Connect()
	if err != nil {
		panic(err)
	}
	defer db.BoltDB.Close()
	log.Info().Msg("successfully connected to database")
	err = models.CreateAdminIfNotExists()
	if err != nil {
		panic(err)
	}
	err = indexer.InitIndexer()
	if err != nil {
		panic(err)
	}
	log.Info().Msg("indexer initialized")
	log.Info().Msg("gemfast server ready")
	err = api.Run()
	if err != nil {
		panic(err)
	}
}
