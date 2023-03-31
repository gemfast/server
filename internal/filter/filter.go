package filter

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/gemfast/server/internal/config"
)

var Filters []string

func InitFilter() error {
	if config.Env.FilterEnabled == "false" {
		return nil
	}
	fp := config.Env.FilterFile
	file, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var filters []string
	for scanner.Scan() {
		filter := scanner.Text()
		if strings.HasPrefix(filter, "#") {
			continue
		}
		filters = append(filters, filter)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	Filters = filters
	return nil
}

func IsAllowed(input string) bool {
	if config.Env.FilterEnabled != "true" {
		return true
	}
	negate := !(config.Env.FilterDefaultDeny == "true")
	for _, f := range Filters {
		m, _ := regexp.MatchString(f, input)

		if m {
			return (m && negate)
		}
	}
	return !negate
}
