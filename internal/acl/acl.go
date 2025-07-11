package acl

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/rs/zerolog/log"
	boltadapter "github.com/speza/casbin-bolt-adapter"
)

type ACL struct {
	Casbin *casbin.Enforcer
	cfg    *config.Config
}

func NewACL(cfg *config.Config, database *db.DB) *ACL {
	var err error

	adapter, err := boltadapter.NewAdapter(database.BoltDB, db.CasbinPoliciesBucket)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create casbin bolt adapter")
	}

	// Use the embedded model.conf for the model (as before)
	modelConf := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load casbin model from string")
	}

	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize the acl with bolt adapter")
	}

	// Enable auto-save for the Bolt adapter
	e.EnableAutoSave(true)

	// Now safe to add/save policies

	// Initialize default ACL policy if empty
	policies, err := e.GetPolicy()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get casbin policy")
	}
	if len(policies) == 0 {
		defaultPolicies := [][]string{
			{"admin", "/*", "*"},
			{"anonymous", "/admin/api/v1/login", "*"},
			{"write", "/admin/api/v1/refresh-token", "*"},
			{"write", "/admin/api/v1/token", "*"},
			{"write", "/admin/api/v1/*", "GET"},
			{"write", fmt.Sprintf("/%s/*", cfg.PrivateGemsNamespace), "*"},
			{"write", "/ui/*", "*"},
			{"read", "/admin/api/v1/refresh-token", "*"},
			{"read", "/admin/api/v1/token", "*"},
			{"read", "/admin/api/v1/*", "GET"},
			{"read", fmt.Sprintf("/%s/*", cfg.PrivateGemsNamespace), "GET"},
			{"read", "/ui/*", "*"},
		}
		for _, p := range defaultPolicies {
			args := make([]interface{}, len(p))
			for i, v := range p {
				args[i] = v
			}
			_, err := e.AddPolicy(args...)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to add default casbin policy")
			}
		}
		err = e.SavePolicy()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to save default casbin policy")
		}
	}

	log.Info().Msg("successfully initialized ACL enforcer with BoltDB adapter")

	return &ACL{Casbin: e, cfg: cfg}
}

func (acl *ACL) Enforce(role string, path string, method string) (bool, error) {
	return acl.Casbin.Enforce(role, path, method)
}
