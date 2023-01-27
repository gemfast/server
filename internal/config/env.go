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
	LogLevel				string `mapstructure:"GEMFAST_LOG_LEVEL"`
	Dir             string `mapstructure:"GEMFAST_DIR"`
	GemDir          string `mapstructure:"GEMFAST_GEM_DIR"`
	DBDir           string `mapstructure:"GEMFAST_DB_DIR"`
	BinPath         string `mapstructure:"GEMFAST_BIN_PATH"`
	URL             string `mapstructure:"GEMFAST_URL"`
	Port            string `mapstructure:"GEMFAST_PORT"`

	// Auth
	AuthMode      string `mapstructure:"GEMFAST_AUTH"`
	AdminPassword string `mapstructure:"GEMFAST_ADMIN_PASSWORD"`
	AddLocalUsers string `mapstructure:"GEMFAST_ADD_LOCAL_USERS"`
}

func configureZeroLog() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func loadEnvVariables() (config *envConfig) {
	viper.BindEnv("GEMFAST_LOG_LEVEL")
	viper.SetDefault("GEMFAST_DIR", "/var/gemfast")
	viper.SetDefault("GEMFAST_GEM_DIR", fmt.Sprintf("%s/gems", viper.Get("GEMFAST_DIR")))
	viper.SetDefault("GEMFAST_DB_DIR", ".")
	viper.SetDefault("GEMFAST_BIN_PATH", "/usr/bin/gemfast")
	viper.SetDefault("GEMFAST_URL", "http://localhost")
	viper.SetDefault("GEMFAST_PORT", 8080)

	viper.SetDefault("GEMFAST_AUTH", "local")
	viper.BindEnv("GEMFAST_ADMIN_PASSWORD")
	viper.BindEnv("GEMFAST_ADD_LOCAL_USERS")

	viper.AutomaticEnv()
	viper.AddConfigPath("$HOME/.gemfast")
	viper.AddConfigPath("/etc/gemfast")
	viper.SetConfigName("gemfast")
	viper.SetConfigType("env")
	viper.ReadInConfig()
	if err := viper.ReadInConfig(); err != nil {
		log.Debug().Err(err).Msg("unable to read in config.env")
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Error().Err(err).Msg("unable to unmarshal config.env")
	}
	return
}
