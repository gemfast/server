package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"

	"github.com/gscho/gemfast/internal/api"
	"github.com/gscho/gemfast/internal/db"
	"github.com/gscho/gemfast/internal/indexer"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the gemfast server",
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	viper.SetEnvPrefix("GEMFAST")
	viper.SetDefault("dir", "/var/gemfast")
	viper.SetDefault("gem_dir", fmt.Sprintf("%s/gems", viper.Get("dir")))
	viper.SetDefault("db_dir", ".")
	viper.SetDefault("auth", "local")
	viper.SetDefault("port", 8080)
	viper.AutomaticEnv()
	viper.SetConfigName(".env")
	viper.AddConfigPath("$HOME/.gemfast")
	viper.AddConfigPath("/etc/gemfast")
	viper.ReadInConfig()
	os.Setenv("PORT", "1")
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
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
