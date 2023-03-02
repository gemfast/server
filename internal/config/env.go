package config

import (
	"fmt"
	"os"
	"os/user"

	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Env envConfig

func InitConfig() {
	configureZeroLog()
	Env = loadEnvVariables()
}

type envConfig struct {
	LogLevel				string `mapstructure:"GEMFAST_LOG_LEVEL"`
	Dir             string `mapstructure:"GEMFAST_DIR"`
	GemDir          string `mapstructure:"GEMFAST_GEM_DIR"`
	DBDir           string `mapstructure:"GEMFAST_DB_DIR"`
	URL             string `mapstructure:"GEMFAST_URL"`
	Port            string `mapstructure:"GEMFAST_PORT"`
	Mirror          string `mapstructure:"GEMFAST_MIRROR"`
	MirrorUpstream  string `mapstructure:"GEMFAST_MIRROR_UPSTREAM"`

	// Auth
	AuthMode      string `mapstructure:"GEMFAST_AUTH"`
	AdminPassword string `mapstructure:"GEMFAST_ADMIN_PASSWORD"`
	AddLocalUsers string `mapstructure:"GEMFAST_ADD_LOCAL_USERS"`
}

func configureZeroLog() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func loadEnvVariables() (config envConfig) {
	var dotEnvMap map[string]string	
	usr, err := user.Current()
	if err != nil {
		log.Error().Err(err).Msg("unable to get current user")
		os.Exit(1)
	}
	homedirConf := fmt.Sprintf("%s/.gemfast/.env", usr.HomeDir)
	if _, err := os.Stat("/etc/gemfast/.env"); err == nil {
		log.Info().Str("configLocation", "/etc/gemfast/.env").Msg("found gemfast config file")
		dotEnvMap, err = godotenv.Read("/etc/gemfast/.env")
	} else if _, err := os.Stat(homedirConf); err == nil {
		log.Info().Str("configLocation", homedirConf).Msg("found gemfast config file")
		dotEnvMap, err = godotenv.Read(homedirConf)
	} else {
		log.Warn().Msg(fmt.Sprintf("unable to find a .env file in /etc/gemfast or %s", homedirConf))
		log.Warn().Msg("using default configuration values")
		dotEnvMap = make(map[string]string)
	}
	setEnvDefaults(dotEnvMap)
	var cfg envConfig
	err = mapstructure.Decode(dotEnvMap, &cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}

func setEnvDefaults(envMap map[string]string) {
	if _, ok := envMap["GEMFAST_DIR"]; !ok {
		envMap["GEMFAST_DIR"] = "/var/gemfast" 
	}
	if _, ok := envMap["GEMFAST_GEM_DIR"]; !ok {
		envMap["GEMFAST_GEM_DIR"] = fmt.Sprintf("%s/gems", envMap["GEMFAST_DIR"])
	}
	if _, ok := envMap["GEMFAST_DB_DIR"]; !ok {
		envMap["GEMFAST_DB_DIR"] = fmt.Sprintf("%s/db", envMap["GEMFAST_DIR"])
	}
	if _, ok := envMap["GEMFAST_URL"]; !ok {
		envMap["GEMFAST_URL"] = "http://localhost" 
	}
	if _, ok := envMap["GEMFAST_PORT"]; !ok {
		envMap["GEMFAST_PORT"] = "8881" 
	}
	if _, ok := envMap["GEMFAST_AUTH"]; !ok {
		envMap["GEMFAST_AUTH"] = "local" 
	}
	if _, ok := envMap["GEMFAST_MIRROR"]; !ok {
		envMap["GEMFAST_MIRROR"] = "" 
	}
	if _, ok := envMap["GEMFAST_MIRROR_UPSTREAM"]; !ok {
		envMap["GEMFAST_MIRROR_UPSTREAM"] = "https://rubygems.org" 
	}
}