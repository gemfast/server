package filter

import (
	"regexp"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/license"
	"github.com/rs/zerolog/log"
)

type RegexFilter struct {
	filters []string
	action  string
	enabled bool
}

func NewFilter(cfg *config.Config, l *license.License) *RegexFilter {
	enabled := false
	if cfg.Filter.Enabled && l.Validated {
		log.Trace().Msg("gem filter enabled")
		enabled = true
	}
	return &RegexFilter{
		filters: cfg.Filter.Regex,
		action:  cfg.Filter.Action,
		enabled: enabled,
	}
}

func (r *RegexFilter) IsAllowed(input string) bool {
	if !r.enabled {
		return true
	}
	negate := !(r.action == "deny")
	for _, f := range r.filters {
		m, _ := regexp.MatchString(f, input)

		if m {
			return (m && negate)
		}
	}
	return !negate
}
