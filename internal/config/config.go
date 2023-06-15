package config

import (
	"fmt"
	"os"
	"os/user"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-password/password"
)

type Config struct {
	Port      int    `hcl:"port,optional"`
	CaddyPort int    `hcl:"caddy_port,optional"`
	LogLevel  string `hcl:"log_level,optional"`
	Dir       string `hcl:"dir,optional"`
	GemDir    string `hcl:"gem_dir,optional"`
	DBDir     string `hcl:"db_dir,optional"`
	URL       string `hcl:"url,optional"`

	TrialMode  bool            `hcl:"trial_mode,optional"`
	LicenseKey string          `hcl:"license_key,optional"`
	Mirrors    []*MirrorConfig `hcl:"mirror,block"`
	Filter     *FilterConfig   `hcl:"filter,block"`
	CVE        *CVEConfig      `hcl:"cve,block"`
	Auth       *AuthConfig     `hcl:"auth,block"`
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
	LocalUsers         []LocalUser `hcl:"users,optional"`
	LocalAuthSecretKey string      `hcl:"secret_key,optional"`
	GitHubClientId     string      `hcl:"github_client_id,optional"`
	GitHubClientSecret string      `hcl:"github_client_secret,optional"`
	GitHubUserOrgs     string      `hcl:"github_user_orgs,optional"`
}

type LocalUser struct {
	Username string `hcl:"username"`
	Password string `hcl:"password"`
	Role     string `hcl:"role"`
}

var Cfg Config

func LoadConfig() {
	cfgFile := os.Getenv("GEMFAST_CONFIG_FILE")
	if cfgFile == "" {
		cfgFileTries := []string{"/etc/gemfast/gemfast.hcl"}
		usr, err := user.Current()
		if err != nil {
			log.Debug().Err(err).Msg("unable to get the current linux user")
		} else {
			cfgFileTries = append(cfgFileTries, fmt.Sprintf("%s/.gemfast/gemfast.hcl", usr.HomeDir))
		}
		for _, f := range cfgFileTries {
			if _, err := os.Stat(f); err == nil {
				cfgFile = f
				log.Info().Str("file", f).Msg(fmt.Sprintf("found gemfast config file at %s", f))
				break
			} else {
				log.Info().Err(err).Msg(fmt.Sprintf("unable to find a gemfast.hcl file at %s", f))
			}
		}

		if cfgFile == "" {
			log.Info().Err(err).Msg(fmt.Sprintf("unable to find a gemfast.hcl file at any of %v", cfgFileTries))
			log.Warn().Msg("using default configuration values")
			Cfg = Config{}
			setDefaultServerConfig(&Cfg)
			return
		}
	}
	err := hclsimple.DecodeFile(cfgFile, nil, &Cfg)
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("failed to load configuration file %s", cfgFile))
		os.Exit(1)
	}
	setDefaultServerConfig(&Cfg)
	setDefaultMirrorConfig(&Cfg)
	setDefaultAuthConfig(&Cfg)
	setDefaultFilterConfig(&Cfg)
	setDefaultCVEConfig(&Cfg)
}

func setDefaultServerConfig(c *Config) {
	if c.Port == 0 {
		c.Port = 2020
	}
	if c.CaddyPort == 0 {

		c.CaddyPort = 443
	}
	if c.URL == "" {
		c.URL = fmt.Sprintf("https://localhost:%d", c.CaddyPort)
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.Dir == "" {
		c.Dir = "/var/gemfast"
	}
	if c.GemDir == "" {
		c.GemDir = fmt.Sprintf("%s/gems", c.Dir)
	}
	if c.DBDir == "" {
		c.DBDir = fmt.Sprintf("%s/gems", c.Dir)
	}
}

func setDefaultMirrorConfig(c *Config) {
	if (c.Mirrors == nil) || (len(c.Mirrors) == 0) {
		c.Mirrors = []*MirrorConfig{{
			Enabled:  true,
			Upstream: "https://rubygems.org",
		}}
	}
}

func setDefaultAuthConfig(c *Config) {
	if c.Auth == nil {
		pw, err := password.Generate(64, 10, 0, false, true)
		if err != nil {
			log.Error().Err(err).Msg("unable to generate a random secret key for local auth")
			os.Exit(1)
		}
		c.Auth = &AuthConfig{
			Type:               "local",
			BcryptCost:         10,
			AllowAnonymousRead: true,
			DefaultUserRole:    "read",
			LocalAuthSecretKey: pw,
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
	if c.Auth.LocalAuthSecretKey == "" {
		pw, err := password.Generate(64, 10, 0, false, true)
		if err != nil {
			log.Error().Err(err).Msg("unable to generate a random secret key for local auth")
			os.Exit(1)
		}
		c.Auth.LocalAuthSecretKey = pw
	}
}

func setDefaultFilterConfig(c *Config) {
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
		c.Filter.Regex = []string{}
	}
}

func setDefaultCVEConfig(c *Config) {
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

func TrialMode() {
	if Cfg.TrialMode {
		Cfg.Auth = &AuthConfig{
			Type: "none",
		}
		Cfg.Mirrors = []*MirrorConfig{{
			Enabled: false,
		}}
		Cfg.Filter = &FilterConfig{
			Enabled: false,
		}
		Cfg.CVE = &CVEConfig{
			Enabled: false,
		}
	}
}
