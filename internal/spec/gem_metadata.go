package spec

import "fmt"

type NestedGemRequirement interface{}

type VersionContraint struct {
	Constraint string `json:"constraint"`
	Version    string `json:"version"`
}

type GemRequirement struct {
	Requirements        []NestedGemRequirement `yaml:"requirements"`
	VersionRequirements []NestedGemRequirement `yaml:"version_requirements"`
	VersionConstraints  []VersionContraint
}

type GemDependency struct {
	Name        string         `yaml:"name"`
	Prerelease  bool           `yaml:"prerelease"`
	Type        string         `yaml:"type"`
	Requirement GemRequirement `yaml:"requirement"`
}

type Email interface{}

type GemMetadata struct {
	Name     string `yaml:"name"`
	Platform string `yaml:"platform"`
	Version  struct {
		Version string `yaml:"version"`
	}
	Authors                 []string `yaml:"authors"`
	Emails                  []string
	Email                   Email           `yaml:"email"`
	Summary                 string          `yaml:"summary"`
	Description             string          `yaml:"description"`
	Homepage                string          `yaml:"homepage"`
	SpecVersion             int             `yaml:"specification_version"`
	RequirePaths            []string        `yaml:"require_paths"`
	Licenses                []string        `yaml:"licenses"`
	RubygemsVersion         string          `yaml:"rubygems_version"`
	Dependencies            []GemDependency `yaml:"dependencies"`
	RequiredRubyVersion     GemRequirement  `yaml:"required_ruby_version"`
	RequiredRubyGemsVersion GemRequirement  `yaml:"required_rubygems_version"`
}

func (gm *GemMetadata) NumInstanceVars() (int, error) {
	ivarCount := 15
	if gm.Name == "" || len(gm.Authors) == 0 || gm.Version.Version == "" {
		return -1, fmt.Errorf("missing required field in gem metadata")
	}
	if gm.Summary == "" {
		ivarCount -= 1
	}
	if len(gm.Emails) == 0 {
		ivarCount -= 1
	}
	if gm.Description == "" {
		ivarCount -= 1
	}
	if gm.Homepage == "" {
		ivarCount -= 1
	}
	if gm.SpecVersion == 0 {
		ivarCount -= 1
	}
	if len(gm.RequirePaths) == 0 {
		ivarCount -= 1
	}
	if len(gm.Licenses) == 0 {
		ivarCount -= 1
	}
	if gm.RubygemsVersion == "" {
		ivarCount -= 1
	}
	if len(gm.Dependencies) == 0 {
		ivarCount -= 1
	}
	return ivarCount, nil
}
