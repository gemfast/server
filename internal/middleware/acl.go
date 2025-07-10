package middleware

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/casbin/casbin/v2"
	"github.com/gemfast/server/internal/config"
	"github.com/rs/zerolog/log"
)

//go:embed model.conf
var embeddedAuthModel []byte

//go:embed acl.csv
var embeddedACL []byte

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
		policyPath = fmt.Sprintf("%s/acl.csv", cfg.Dir)
		err = os.WriteFile(policyPath, embeddedACL, 0644)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to write embedded acl to temp file")
		}
	}

	if cfg.AuthModelPath != "" {
		authPath, err = filepath.Abs(cfg.AuthModelPath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get absolute path for auth_model")
		}
	} else {
		authPath = fmt.Sprintf("%s/model.conf", cfg.Dir)
		err = os.WriteFile(authPath, embeddedAuthModel, 0644)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to write embedded auth model to temp file")
		}
	}

	if policyPath == "" || authPath == "" {
		log.Fatal().Err(fmt.Errorf("unable to locate model.conf and acl.csv")).Msg("failed to find acl files")
	}
	acl, err := casbin.NewEnforcer(authPath, policyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize the acl")
	}
	log.Info().Str("detail", policyPath).Str("detail", authPath).Msg("successfully initialized ACL enforcer")

	return &ACL{casbin: acl, cfg: cfg}
}

func (acl *ACL) Enforce(role string, path string, method string) (bool, error) {
	return acl.casbin.Enforce(role, path, method)
}
