package filter

import (
	"testing"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/license"
)

func TestIsAllowed(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Filter = &config.FilterConfig{
		Enabled: true,
		Regex:   []string{"webmock*"},
		Action:  "deny",
	}
	f := NewFilter(cfg, &license.License{Validated: true})
	allowed := f.IsAllowed("webmock-3.18.1.gem")
	if allowed {
		t.Errorf("DenyList: expected webmock-3.18.1.gem to be denied")
	}
	allowed = f.IsAllowed("tty-box-0.7.0.gem")
	if !allowed {
		t.Errorf("DenyList: expected tty-box-0.7.0.gem to be allowed")
	}

	cfg.Filter.Action = "allow"
	f = NewFilter(cfg, &license.License{Validated: true})
	allowed = f.IsAllowed("webmock-3.18.1.gem")
	if !allowed {
		t.Errorf("AllowList: expected webmock-3.18.1.gem to be allowed")
	}
	allowed = f.IsAllowed("tty-box-0.7.0.gem")
	if allowed {
		t.Errorf("AllowList: expected tty-box-0.7.0.gem to be denied")
	}
}
