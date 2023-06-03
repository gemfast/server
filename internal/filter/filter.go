package filter

import (
	"regexp"

	"github.com/gemfast/server/internal/config"
)

var Filters []string

func InitFilter() error {
	if !config.Cfg.Filter.Enabled {
		return nil
	}
	filters := config.Cfg.Filter.Regex
	Filters = filters
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
