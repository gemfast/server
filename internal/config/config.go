package config

import (
	"fmt"
	"os"
	"os/user"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-password/password"
)

type Config struct {
	Port           int    `hcl:"port,optional"`
	LogLevel       string `hcl:"log_level,optional"`
	Dir            string `hcl:"dir,optional"`
	GemDir         string `hcl:"gem_dir,optional"`
	DBDir          string `hcl:"db_dir,optional"`
	ACLPath        string `hcl:"acl_path,optional"`
	AuthModelPath  string `hcl:"auth_model_path,optional"`
	PrivateGemsURL string `hcl:"private_gems_url,optional"`

	TrialMode   bool            `hcl:"trial_mode,optional"`
	LicenseKey  string          `hcl:"license_key,optional"`
	CaddyConfig *CaddyConfig    `hcl:"caddy,block"`
	Mirrors     []*MirrorConfig `hcl:"mirror,block"`
	Filter      *FilterConfig   `hcl:"filter,block"`
	CVE         *CVEConfig      `hcl:"cve,block"`
	Auth        *AuthConfig     `hcl:"auth,block"`
}

type CaddyConfig struct {
	AdminAPIEnabled bool   `hcl:"admin_api_enabled,optional"`
	MetricsDisabled bool   `hcl:"metrics_disabled,optional"`
	Host            string `hcl:"host,optional"`
	Port            int    `hcl:"port,optional"`
}

type MirrorConfig struct {
	Enabled  bool   `hcl:"enabled,optional"`
	Upstream string `hcl:"upstream,label"`
}

type FilterConfig struct {
	Enabled bool     `hcl:"enabled,optional"`
	Action  string   `hcl:"action,optional"`
	Regex   []string `hcl:"regex"`
}

type CVEConfig struct {
	Enabled           bool   `hcl:"enabled,optional"`
	MaxSeverity       string `hcl:"max_severity,optional"`
	RubyAdvisoryDBDir string `hcl:"ruby_advisory_db_dir,optional"`
}

type AuthConfig struct {
	Type               string      `hcl:"type,label"`
	BcryptCost         int         `hcl:"bcrypt_cost,optional"`
	AdminPassword      string      `hcl:"admin_password,optional"`
	DefaultUserRole    string      `hcl:"default_user_role,optional"`
	AllowAnonymousRead bool        `hcl:"allow_anonymous_read,optional"`
	LocalUsers         []LocalUser `hcl:"user,block"`
	JWTSecretKey       string      `hcl:"secret_key,optional"`
	JWTSecretKeyPath   string      `hcl:"secret_key_path,optional"`
	GitHubClientId     string      `hcl:"github_client_id,optional"`
	GitHubClientSecret string      `hcl:"github_client_secret,optional"`
	GitHubUserOrgs     []string    `hcl:"github_user_orgs,optional"`
}

type LocalUser struct {
	Username string `hcl:"username"`
	Password string `hcl:"password"`
	Role     string `hcl:"role,optional"`
}

func NewConfig() *Config {
	cfg := Config{}
	cfgFile := os.Getenv("GEMFAST_CONFIG_FILE")
	if cfgFile == "" {
		cfgFileTries := []string{"/etc/gemfast/gemfast.hcl"}
		usr, err := user.Current()
		if err != nil {
			log.Warn().Err(err).Msg("unable to get the current linux user")
		} else {
			cfgFileTries = append(cfgFileTries, fmt.Sprintf("%s/.gemfast/gemfast.hcl", usr.HomeDir))
		}
		for _, f := range cfgFileTries {
			if _, err := os.Stat(f); err == nil {
				cfgFile = f
				log.Info().Str("detail", f).Msg("found gemfast config file")
				break
			}
		}

		if cfgFile == "" {
			log.Warn().Err(err).Msg(fmt.Sprintf("unable to find a gemfast.hcl file at any of %v", cfgFileTries))
			log.Warn().Msg("using default configuration values")
			cfg.setDefaultConfig()
			return &cfg
		}
	}
	err := hclsimple.DecodeFile(cfgFile, nil, &cfg)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("failed to load configuration file %s", cfgFile))
		os.Exit(1)
	}
	cfg.setDefaultConfig()
	return &cfg
}

func (c *Config) setDefaultConfig() {
	c.setDefaultServerConfig()
	c.setDefaultCaddyConfig()
	c.setDefaultMirrorConfig()
	c.setDefaultAuthConfig()
	c.setDefaultFilterConfig()
	c.setDefaultCVEConfig()
}

func (c *Config) setDefaultServerConfig() {
	if c.Port == 0 {
		c.Port = 2020
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	configureLogLevel(c.LogLevel)
	if c.Dir == "" {
		c.Dir = "/var/gemfast"
	}
	if c.GemDir == "" {
		c.GemDir = fmt.Sprintf("%s/gems", c.Dir)
	}
	if c.DBDir == "" {
		c.DBDir = fmt.Sprintf("%s/db", c.Dir)
	}
	if c.PrivateGemsURL == "" {
		c.PrivateGemsURL = "/private"
	}
}

func (c *Config) setDefaultCaddyConfig() {
	if c.CaddyConfig == nil {
		c.CaddyConfig = &CaddyConfig{
			AdminAPIEnabled: false,
			MetricsDisabled: false,
			Host:            "https://localhost:443",
			Port:            443,
		}
		return
	}
	if c.CaddyConfig.Port == 0 {
		c.CaddyConfig.Port = 443
	}
	if c.CaddyConfig.Host == "" {
		c.CaddyConfig.Host = fmt.Sprintf("https://localhost:%d", c.CaddyConfig.Port)
	}
}

func configureLogLevel(ll string) {
	level, err := zerolog.ParseLevel(ll)
	if err != nil {
		log.Error().Err(err).Msg("invalid log_level, expecting one of trace, debug, info, warn, error, fatal, panic")
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Info().Str("detail", level.String()).Msg("set global log level")
}

func (c *Config) setDefaultMirrorConfig() {
	if (c.Mirrors == nil) || (len(c.Mirrors) == 0) {
		c.Mirrors = []*MirrorConfig{{
			Enabled:  true,
			Upstream: "https://rubygems.org",
		}}
	}
}

func readJWTSecretKeyFromPath(keyPath string) string {
	if _, err := os.Stat(keyPath); err == nil {
		log.Info().Str("detail", keyPath).Msg("using JWT secret key from file")
		key, err := os.ReadFile(keyPath)
		if err != nil {
			log.Error().Err(err).Msg("unable to read JWT secret key from file")
			os.Exit(1)
		}
		return string(key)
	}
	log.Info().Msg("generating a new JWT secret key")
	pw, err := password.Generate(64, 10, 0, false, true)
	if err != nil {
		log.Error().Err(err).Msg("unable to generate a new jwt secret key")
		os.Exit(1)
	}
	file, err := os.Create(keyPath)
	if err != nil {
		log.Error().Err(err).Msg("unable to create JWT secret key file")
		os.Exit(1)
	}
	defer file.Close()
	_, err = file.WriteString(pw)
	if err != nil {
		log.Error().Err(err).Msg("unable to write JWT secret key to file")
		log.Error().Msg("JWT secret key will not persist after server is stopped")
		os.Remove(keyPath)
	}
	return pw
}

func (c *Config) setDefaultAuthConfig() {
	defaultJWTSecretKeyPath := "/opt/gemfast/etc/gemfast/.jwt_secret_key"
	if c.Auth == nil {
		c.Auth = &AuthConfig{
			Type:               "local",
			BcryptCost:         10,
			AllowAnonymousRead: false,
			DefaultUserRole:    "read",
			JWTSecretKeyPath:   defaultJWTSecretKeyPath,
			JWTSecretKey:       readJWTSecretKeyFromPath(defaultJWTSecretKeyPath),
		}
		return
	}
	if c.Auth.Type == "" {
		c.Auth.Type = "local"
	}
	if c.Auth.BcryptCost == 0 {
		c.Auth.BcryptCost = 10
	}
	if c.Auth.DefaultUserRole == "" {
		c.Auth.DefaultUserRole = "read"
	}
	if c.Auth.JWTSecretKeyPath == "" {
		c.Auth.JWTSecretKeyPath = defaultJWTSecretKeyPath
	}
	if c.Auth.JWTSecretKey == "" {
		readJWTSecretKeyFromPath(defaultJWTSecretKeyPath)
	}
}

func (c *Config) setDefaultFilterConfig() {
	if c.Filter == nil {
		c.Filter = &FilterConfig{
			Enabled: false,
			Action:  "allow",
			Regex:   []string{},
		}
		return
	}
	if c.Filter.Action == "" {
		c.Filter.Action = "allow"
	}
	if c.Filter.Regex == nil {
		c.Filter.Regex = []string{"*"}
	}
}

func (c *Config) setDefaultCVEConfig() {
	if c.CVE == nil {
		c.CVE = &CVEConfig{
			Enabled:           false,
			MaxSeverity:       "high",
			RubyAdvisoryDBDir: "/opt/gemfast/share/ruby-advisory-db",
		}
		return
	}
	if c.CVE.MaxSeverity == "" {
		c.CVE.MaxSeverity = "high"
	}
	if c.CVE.RubyAdvisoryDBDir == "" {
		c.CVE.RubyAdvisoryDBDir = "/opt/gemfast/share/ruby-advisory-db"
	}
}
