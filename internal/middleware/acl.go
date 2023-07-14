package middleware

import (
	"fmt"
	"path/filepath"

	"github.com/casbin/casbin/v2"
	"github.com/gemfast/server/internal/config"
	u "github.com/gemfast/server/internal/utils"
	"github.com/rs/zerolog/log"
)

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

	if policyPath == "" || authPath == "" {
		log.Fatal().Err(fmt.Errorf("unable to locate auth_model and gemfast_acl")).Msg("failed to find acl files")
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
