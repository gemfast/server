package config

import (
	"fmt"
	"os"
	"os/user"

	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-password/password"
)

var Env envConfig

func InitConfig() {
	Env = loadEnvVariables()
	configureZeroLog()
}

type envConfig struct {
	LogLevel          string `mapstructure:"GEMFAST_LOG_LEVEL"`
	Dir               string `mapstructure:"GEMFAST_DIR"`
	GemDir            string `mapstructure:"GEMFAST_GEM_DIR"`
	DBDir             string `mapstructure:"GEMFAST_DB_DIR"`
	URL               string `mapstructure:"GEMFAST_URL"`
	Port              string `mapstructure:"GEMFAST_PORT"`
	MirrorEnabled     string `mapstructure:"GEMFAST_MIRROR_ENABLED"`
	MirrorUpstream    string `mapstructure:"GEMFAST_MIRROR_UPSTREAM"`
	FilterEnabled     string `mapstructure:"GEMFAST_FILTER_ENABLED"`
	FilterDefaultDeny string `mapstructure:"GEMFAST_FILTER_DEFAULT_DENY"`
	FilterFile        string `mapstructure:"GEMFAST_FILTER_FILE"`
	CVEFilterEnabled  string `mapstructure:"GEMFAST_CVE_FILTER_ENABLED"`
	MaxCVESeverity    string `mapstructure:"GEMFAST_CVE_MAX_SEVERITY"`
	RubyAdvisoryDBDir string `mapstructure:"GEMFAST_RUBY_ADVISORY_DB_DIR"`

	// Auth
	AuthMode               string `mapstructure:"GEMFAST_AUTH"`
	BcryptDefaultCost      string `mapstructure:"GEMFAST_BCRYPT_DEFAULT_COST"`
	AdminPassword          string `mapstructure:"GEMFAST_ADMIN_PASSWORD"`
	AddLocalUsers          string `mapstructure:"GEMFAST_ADD_LOCAL_USERS"`
	LocalUsersDefaultRole  string `mapstructure:"GEMFAST_LOCAL_USERS_DEFAULT_ROLE"`
	LocalAuthSecretKey     string `mapstructure:"GEMFAST_LOCAL_AUTH_SECRET_KEY"`
	AllowAnonymousRead     string `mapstructure:"GEMFAST_ALLOW_ANONYMOUS_READ"`
	GitHubUsersDefaultRole string `mapstructure:"GEMFAST_GITHUB_USERS_DEFAULT_ROLE"`
	GitHubClientId         string `mapstructure:"GEMFAST_GITHUB_CLIENT_ID"`
	GitHubClientSecret     string `mapstructure:"GEMFAST_GITHUB_CLIENT_SECRET"`
	GitHubUserOrgs         string `mapstructure:"GEMFAST_GITHUB_USER_ORGS"`

	//License
	GemfastLicenseKey string `mapstructure:"GEMFAST_LICENSE_KEY"`
	GemfastTrialMode  string `mapstructure:"GEMFAST_TRIAL_MODE"`
}

func configureZeroLog() {
	ll, err := zerolog.ParseLevel(Env.LogLevel)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse GEMFAST_LOG_LEVEL to a valid zerolog level")
		ll = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(ll)
	log.Info().Str("detail", ll.String()).Msg("set global log level")
}

func loadEnvVariables() (config envConfig) {
	var dotEnvMap map[string]string
	usr, err := user.Current()
	if err != nil {
		log.Error().Err(err).Msg("unable to get the current linux user. Please contact support: https://gemfast.io/support")
		os.Exit(1)
	}
	homedirConf := fmt.Sprintf("%s/.gemfast/.env", usr.HomeDir)
	if _, err := os.Stat("/etc/gemfast/.env"); err == nil {
		log.Info().Str("detail", "/etc/gemfast/.env").Msg("found gemfast config file")
		dotEnvMap, _ = godotenv.Read("/etc/gemfast/.env")
	} else if _, err := os.Stat(homedirConf); err == nil {
		log.Info().Str("detail", homedirConf).Msg("found gemfast config file")
		dotEnvMap, _ = godotenv.Read(homedirConf)
	} else {
		log.Warn().Msg(fmt.Sprintf("unable to find a .env file in /etc/gemfast or %s", homedirConf))
		log.Warn().Msg("using default configuration values")
		dotEnvMap = make(map[string]string)
	}
	setEnvDefaults(dotEnvMap)
	setExportedEnv(dotEnvMap)
	var cfg envConfig
	err = mapstructure.Decode(dotEnvMap, &cfg)
	if err != nil {
		log.Error().Err(err).Msg("Unable to decode config into a mapstructure. Please contact support: https://gemfast.io/support")
		os.Exit(1)
	}
	return cfg
}

func setEnvDefaults(envMap map[string]string) {
	if _, ok := envMap["GEMFAST_LOG_LEVEL"]; !ok {
		envMap["GEMFAST_LOG_LEVEL"] = "info"
	}
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
		envMap["GEMFAST_PORT"] = "2020"
	}
	if _, ok := envMap["GEMFAST_AUTH"]; !ok {
		envMap["GEMFAST_AUTH"] = "local"
	}
	if _, ok := envMap["GEMFAST_BCRYPT_DEFAULT_COST"]; !ok {
		envMap["GEMFAST_BCRYPT_DEFAULT_COST"] = "10"
	}
	if _, ok := envMap["GEMFAST_MIRROR_ENABLED"]; !ok {
		envMap["GEMFAST_MIRROR_ENABLED"] = "true"
	}
	if _, ok := envMap["GEMFAST_MIRROR_UPSTREAM"]; !ok {
		envMap["GEMFAST_MIRROR_UPSTREAM"] = "https://rubygems.org"
	}
	if _, ok := envMap["GEMFAST_FILTER_ENABLED"]; !ok {
		envMap["GEMFAST_FILTER_ENABLED"] = "false"
	}
	if _, ok := envMap["GEMFAST_FILTER_DEFAULT_DENY"]; !ok {
		envMap["GEMFAST_FILTER_DEFAULT_DENY"] = "true"
	}
	if _, ok := envMap["GEMFAST_FILTER_FILE"]; !ok {
		envMap["GEMFAST_FILTER_FILE"] = "/etc/gemfast/filter.conf"
	}
	if _, ok := envMap["GEMFAST_CVE_FILTER_ENABLED"]; !ok {
		envMap["GEMFAST_CVE_FILTER_ENABLED"] = "true"
	}
	if _, ok := envMap["GEMFAST_CVE_MAX_SEVERITY"]; !ok {
		envMap["GEMFAST_CVE_MAX_SEVERITY"] = "high"
	}
	if _, ok := envMap["GEMFAST_RUBY_ADVISORY_DB_DIR"]; !ok {
		envMap["GEMFAST_RUBY_ADVISORY_DB_DIR"] = "/opt/gemfast/share/ruby-advisory-db/gems"
	}
	if _, ok := envMap["GEMFAST_LOCAL_USERS_DEFAULT_ROLE"]; !ok {
		envMap["GEMFAST_LOCAL_USERS_DEFAULT_ROLE"] = "read"
	}
	if _, ok := envMap["GEMFAST_GITHUB_USERS_DEFAULT_ROLE"]; !ok {
		envMap["GEMFAST_GITHUB_USERS_DEFAULT_ROLE"] = "read"
	}
	if _, ok := envMap["GEMFAST_LOCAL_AUTH_SECRET_KEY"]; !ok {
		s, _ := password.Generate(64, 10, 0, false, false)
		envMap["GEMFAST_LOCAL_AUTH_SECRET_KEY"] = s
	}
	if _, ok := envMap["GEMFAST_ALLOW_ANONYMOUS_READ"]; !ok {
		envMap["GEMFAST_ALLOW_ANONYMOUS_READ"] = "false"
	}
	if _, ok := envMap["GEMFAST_TRIAL_MODE"]; !ok {
		envMap["GEMFAST_TRIAL_MODE"] = "true"
	}
}

func setExportedEnv(envMap map[string]string) {
	envMap["GEMFAST_LOG_LEVEL"] = getEnv("GEMFAST_LOG_LEVEL", envMap["GEMFAST_LOG_LEVEL"])
	envMap["GEMFAST_DIR"] = getEnv("GEMFAST_DIR", envMap["GEMFAST_DIR"])
	envMap["GEMFAST_GEM_DIR"] = getEnv("GEMFAST_GEM_DIR", envMap["GEMFAST_GEM_DIR"])
	envMap["GEMFAST_DB_DIR"] = getEnv("GEMFAST_DB_DIR", envMap["GEMFAST_DB_DIR"])
	envMap["GEMFAST_URL"] = getEnv("GEMFAST_URL", envMap["GEMFAST_URL"])
	envMap["GEMFAST_PORT"] = getEnv("GEMFAST_PORT", envMap["GEMFAST_PORT"])
	envMap["GEMFAST_AUTH"] = getEnv("GEMFAST_AUTH", envMap["GEMFAST_AUTH"])
	envMap["GEMFAST_BCRYPT_DEFAULT_COST"] = getEnv("GEMFAST_BCRYPT_DEFAULT_COST", envMap["GEMFAST_BCRYPT_DEFAULT_COST"])
	envMap["GEMFAST_MIRROR_ENABLED"] = getEnv("GEMFAST_MIRROR_ENABLED", envMap["GEMFAST_MIRROR_ENABLED"])
	envMap["GEMFAST_MIRROR_UPSTREAM"] = getEnv("GEMFAST_MIRROR_UPSTREAM", envMap["GEMFAST_MIRROR_UPSTREAM"])
	envMap["GEMFAST_FILTER_ENABLED"] = getEnv("GEMFAST_FILTER_ENABLED", envMap["GEMFAST_FILTER_ENABLED"])
	envMap["GEMFAST_FILTER_DEFAULT_DENY"] = getEnv("GEMFAST_FILTER_DEFAULT_DENY", envMap["GEMFAST_FILTER_DEFAULT_DENY"])
	envMap["GEMFAST_FILTER_FILE"] = getEnv("GEMFAST_FILTER_FILE", envMap["GEMFAST_FILTER_FILE"])
	envMap["GEMFAST_CVE_FILTER_ENABLED"] = getEnv("GEMFAST_CVE_FILTER_ENABLED", envMap["GEMFAST_CVE_FILTER_ENABLED"])
	envMap["GEMFAST_CVE_MAX_SEVERITY"] = getEnv("GEMFAST_CVE_MAX_SEVERITY", envMap["GEMFAST_CVE_MAX_SEVERITY"])
	envMap["GEMFAST_RUBY_ADVISORY_DB_DIR"] = getEnv("GEMFAST_RUBY_ADVISORY_DB_DIR", envMap["GEMFAST_RUBY_ADVISORY_DB_DIR"])
	envMap["GEMFAST_LOCAL_USERS_DEFAULT_ROLE"] = getEnv("GEMFAST_LOCAL_USERS_DEFAULT_ROLE", envMap["GEMFAST_LOCAL_USERS_DEFAULT_ROLE"])
	envMap["GEMFAST_LOCAL_AUTH_SECRET_KEY"] = getEnv("GEMFAST_LOCAL_AUTH_SECRET_KEY", envMap["GEMFAST_LOCAL_AUTH_SECRET_KEY"])
	envMap["GEMFAST_ALLOW_ANONYMOUS_READ"] = getEnv("GEMFAST_ALLOW_ANONYMOUS_READ", envMap["GEMFAST_ALLOW_ANONYMOUS_READ"])
	envMap["GEMFAST_TRIAL_MODE"] = getEnv("GEMFAST_TRIAL_MODE", envMap["GEMFAST_TRIAL_MODE"])
	envMap["GEMFAST_ADMIN_PASSWORD"] = getEnv("GEMFAST_ADMIN_PASSWORD", envMap["GEMFAST_ADMIN_PASSWORD"])
	envMap["GEMFAST_ADD_LOCAL_USERS"] = getEnv("GEMFAST_ADD_LOCAL_USERS", envMap["GEMFAST_ADD_LOCAL_USERS"])
	envMap["GEMFAST_LOCAL_USERS_DEFAULT_ROLE"] = getEnv("GEMFAST_LOCAL_USERS_DEFAULT_ROLE", envMap["GEMFAST_LOCAL_USERS_DEFAULT_ROLE"])
	envMap["GEMFAST_GITHUB_USERS_DEFAULT_ROLE"] = getEnv("GEMFAST_GITHUB_USERS_DEFAULT_ROLE", envMap["GEMFAST_GITHUB_USERS_DEFAULT_ROLE"])
	envMap["GEMFAST_GITHUB_USER_ORGS"] = getEnv("GEMFAST_GITHUB_USER_ORGS", envMap["GEMFAST_GITHUB_USER_ORGS"])
	envMap["GEMFAST_LOCAL_AUTH_SECRET_KEY"] = getEnv("GEMFAST_LOCAL_AUTH_SECRET_KEY", envMap["GEMFAST_LOCAL_AUTH_SECRET_KEY"])
	envMap["GEMFAST_LICENSE_KEY"] = getEnv("GEMFAST_LICENSE_KEY", envMap["GEMFAST_LICENSE_KEY"])
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}
