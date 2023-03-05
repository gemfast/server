package filter

import (
	"github.com/gemfast/server/internal/config"
	"testing"
)

func TestIsAllowed(t *testing.T) {
	config.Env.FilterEnabled = "true"
	config.Env.FilterFile = "../../test/fixtures/filter/filter_1.conf"
	config.Env.FilterDefaultDeny = "true"
	InitFilter()
	allowed := IsAllowed("webmock-3.18.1.gem")
	if allowed {
		t.Errorf("DenyList: expected webmock-3.18.1.gem to be denied")
	}
	allowed = IsAllowed("tty-box-0.7.0.gem")
	if !allowed {
		t.Errorf("DenyList: expected tty-box-0.7.0.gem to be allowed")
	}

	config.Env.FilterDefaultDeny = "false"
	allowed = IsAllowed("webmock-3.18.1.gem")
	if !allowed {
		t.Errorf("AllowList: expected webmock-3.18.1.gem to be allowed")
	}
	config.Env.FilterDefaultDeny = "false"
	allowed = IsAllowed("tty-box-0.7.0.gem")
	if allowed {
		t.Errorf("AllowList: expected tty-box-0.7.0.gem to be denied")
	}
}
