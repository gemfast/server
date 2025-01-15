package middleware

import (
	"embed"
	"path/filepath"

	"github.com/casbin/casbin/v2"
	"github.com/gemfast/server/internal/config"
	u "github.com/gemfast/server/internal/utils"
	casbin_fs_adapter "github.com/naucon/casbin-fs-adapter"
	"github.com/rs/zerolog/log"
)

//go:embed casbin/auth_model.conf casbin/gemfast_acl.csv
var fs embed.FS

type ACL struct {
	casbin *casbin.Enforcer
	cfg    *config.Config
}

func NewACL(cfg *config.Config) *ACL {
	var policyPath string
	var authPath string
	var err error

	if cfg.ACLPath != "" {
		policyPath, err = filepath.Abs(cfg.ACLPath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get absolute path for acl")
		}
	} else {
		for _, path := range []string{"/opt/gemfast/etc/gemfast/gemfast_acl.csv", "gemfast_acl.csv"} {
			exists, _ := u.FileExists(path)
			if exists {
				policyPath = path
				break
			}
		}
	}

	if cfg.AuthModelPath != "" {
		authPath, err = filepath.Abs(cfg.AuthModelPath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get absolute path for auth_model")
		}
	} else {
		for _, path := range []string{"/opt/gemfast/etc/gemfast/auth_model.conf", "auth_model.conf"} {
			exists, _ := u.FileExists(path)
			if exists {
				authPath = path
				break
			}
		}
	}

	if policyPath == "" && authPath == "" {
		model, err := casbin_fs_adapter.NewModel(fs, "auth_model.conf")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load casbin model")
		}
		policies := casbin_fs_adapter.NewAdapter(fs, "gemfast_acl.csv")
		enforcer, err := casbin.NewEnforcer(model, policies)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create casbin enforcer")
		}
		return &ACL{casbin: enforcer, cfg: cfg}
	}
	acl, err := casbin.NewEnforcer(authPath, policyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize the acl")
	} else {
		log.Info().Str("detail", policyPath).Msg("successfully initialized ACL enforcer")
	}
	return &ACL{casbin: acl, cfg: cfg}
}

func (acl *ACL) Enforce(role string, path string, method string) (bool, error) {
	return acl.casbin.Enforce(role, path, method)
}
