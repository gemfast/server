package main

import (
	"github.com/gscho/gemfast/internal/api"
	"github.com/gscho/gemfast/internal/db"
	"github.com/gscho/gemfast/internal/indexer"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("GEMFAST")
	viper.BindEnv("DIR")
	viper.SetDefault("dir", "/var/gemfast")
	viper.AutomaticEnv()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	err := db.InitDB()
	if err != nil {
		panic(err)
	}
	defer db.DB.Close()
	err = indexer.InitIndexer()
	if err != nil {
		panic(err)
	}
	err = api.Run()
	if err != nil {
		panic(err)
	}
}
