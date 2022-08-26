package main

import (
	"fmt"

	"github.com/gscho/gemfast/internal/api"
	"github.com/gscho/gemfast/internal/db"
	"github.com/gscho/gemfast/internal/indexer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("GEMFAST")
	viper.BindEnv("DIR")
	viper.BindEnv("GEM_DIR")
	viper.SetDefault("dir", "/var/gemfast")
	viper.SetDefault("gem_dir", fmt.Sprintf("%s/gems", viper.Get("dir")))
	viper.AutomaticEnv()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	err := db.Connect()
	if err != nil {
		panic(err)
	}
	log.Info().Msg("successfully connected to database")
	defer db.BoltDB.Close()
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
