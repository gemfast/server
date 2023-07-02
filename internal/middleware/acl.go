package middleware

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/casbin/casbin/v2"
	"github.com/gemfast/server/internal/config"
	u "github.com/gemfast/server/internal/utils"
	"github.com/rs/zerolog/log"
)

var ACL casbin.Enforcer

func InitACL() {
	var policyPath string
	var authPath string
	var err error

	if config.Cfg.ACLPath != "" {
		policyPath, err = filepath.Abs(config.Cfg.ACLPath)
		if err != nil {
			log.Error().Err(err).Msg("failed to get absolute path for acl")
			os.Exit(1)
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

	if config.Cfg.AuthModelPath != "" {
		authPath, err = filepath.Abs(config.Cfg.AuthModelPath)
		if err != nil {
			log.Error().Err(err).Msg("failed to get absolute path for auth_model")
			os.Exit(1)
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
		log.Error().Err(fmt.Errorf("unable to locate auth_model and gemfast_acl")).Msg("failed to find acl files")
		os.Exit(1)
	}
	acl, err := casbin.NewEnforcer(authPath, policyPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize the acl")
		os.Exit(1)
	} else {
		log.Info().Str("detail", policyPath).Msg("successfully initialized ACL enforcer")
	}
	ACL = *acl
}
