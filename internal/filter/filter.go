package filter

import (
	"regexp"

	"github.com/gemfast/server/internal/config"
	"github.com/rs/zerolog/log"
)

var Filters []string

func InitFilter() error {
	filters := config.Cfg.Filter.Regex
	Filters = filters
	log.Info().Msg("gem filter initialized")
	return nil
}

func IsAllowed(input string) bool {
	if !config.Cfg.Filter.Enabled {
		return true
	}
	negate := !(config.Cfg.Filter.Action == "deny")
	for _, f := range Filters {
		m, _ := regexp.MatchString(f, input)

		if m {
			return (m && negate)
		}
	}
	return !negate
}
