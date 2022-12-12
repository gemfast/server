package main

import (
	"fmt"

	gemfast "github.com/gscho/gemfast/cmd/gemfast"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvPrefix("GEMFAST")
	viper.SetDefault("dir", "/var/gemfast")
	viper.SetDefault("gem_dir", fmt.Sprintf("%s/gems", viper.Get("dir")))
	viper.SetDefault("db_dir", ".")
	viper.SetDefault("auth", "local")
	viper.AutomaticEnv()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	gemfast.Execute()
}
