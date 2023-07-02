package filter

import (
	"testing"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/license"
)

func TestIsAllowed(t *testing.T) {
	config.Cfg.Filter = &config.FilterConfig{
		Enabled: true,
		Regex:   []string{"webmock*"},
		Action:  "deny",
	}
	InitFilter(&license.License{Validated: true})
	allowed := IsAllowed("webmock-3.18.1.gem")
	if allowed {
		t.Errorf("DenyList: expected webmock-3.18.1.gem to be denied")
	}
	allowed = IsAllowed("tty-box-0.7.0.gem")
	if !allowed {
		t.Errorf("DenyList: expected tty-box-0.7.0.gem to be allowed")
	}

	config.Cfg.Filter.Action = "allow"
	allowed = IsAllowed("webmock-3.18.1.gem")
	if !allowed {
		t.Errorf("AllowList: expected webmock-3.18.1.gem to be allowed")
	}
	config.Cfg.Filter.Action = "allow"
	allowed = IsAllowed("tty-box-0.7.0.gem")
	if allowed {
		t.Errorf("AllowList: expected tty-box-0.7.0.gem to be denied")
	}
}
