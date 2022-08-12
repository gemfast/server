package main

import (
	"github.com/gscho/gemfast/internal/api"
	"github.com/gscho/gemfast/internal/indexer"
	zl "github.com/rs/zerolog"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("GEMFAST")
	viper.BindEnv("DIR")
	viper.SetDefault("dir", "/var/gemfast")
	viper.AutomaticEnv()
	zl.TimeFieldFormat = zl.TimeFormatUnix
	zl.SetGlobalLevel(zl.DebugLevel)
}

func main() {
	err := indexer.InitIndexer(); if err != nil {
		panic(err)
	}
	err = api.Run(); if err != nil {
		panic(err)
	}
}
