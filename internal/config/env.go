package config

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var Env *envConfig

func InitConfig() {
	configureZeroLog()
	Env = loadEnvVariables()
}

type envConfig struct {
	LocalServerPort string `mapstructure:"LOCAL_SERVER_PORT"`
	SecretKey       string `mapstructure:"SECRET_KEY"`
	Dir string `mapstructure:"GEMFAST_DIR"`
	GemDir string `mapstructure:"GEMFAST_GEM_DIR"`
	DBDir string `mapstructure:"GEMFAST_DB_DIR"`
	Port string `mapstructure:"GEMFAST_PORT"`

	// Auth stuff
	AuthMode string `mapstructure:"GEMFAST_AUTH"`
	AdminPassword string `mapstructure:"GEMFAST_ADMIN_PASSWORD"`
	AddLocalUsers string `mapstructure:"GEMFAST_ADD_LOCAL_USERS"`
}

func configureZeroLog() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func loadEnvVariables() (config *envConfig) {
	viper.SetDefault("GEMFAST_DIR", "/var/gemfast")
	viper.SetDefault("GEMFAST_GEM_DIR", fmt.Sprintf("%s/gems", viper.Get("dir")))
	viper.SetDefault("GEMFAST_DB_DIR", ".")
	viper.SetDefault("GEMFAST_AUTH", "local")
	viper.SetDefault("GEMFAST_PORT", 8080)
	viper.AutomaticEnv()
	viper.AddConfigPath("$HOME/.gemfast")
	viper.AddConfigPath("/etc/gemfast")
	viper.SetConfigName("config")
	viper.SetConfigType("env")
	viper.ReadInConfig()
	if err := viper.ReadInConfig(); err != nil {
		log.Error().Err(err).Msg("unable to read in config.env")
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Error().Err(err).Msg("unable to unmarshal config.env")
	}
	return
}