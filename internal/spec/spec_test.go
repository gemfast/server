package spec

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	// "gopkg.in/yaml.v3"
)

type SpecTestSuite struct {
	suite.Suite
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(SpecTestSuite))
}

func (suite *SpecTestSuite) TestParseGemMetadata() {
	res, err := os.ReadFile("../../test/devise-metadata.yml")
	if err != nil {
		suite.FailNow(fmt.Sprintf("failed to read devise-metadata.yml: %s", err))
	}
	metadata, err := ParseGemMetadata([]byte(res))
	suite.Nil(err)
	suite.NotNil(metadata)
	suite.Equal("devise", metadata.Name)
	suite.Equal("4.7.1", metadata.Version.Version)
	suite.Equal("ruby", metadata.Platform)
	suite.Equal("Flexible authentication solution for Rails with Warden", metadata.Summary)
	suite.Equal(">=", metadata.RequiredRubyVersion.VersionConstraints[0].Constraint)
	suite.Equal("2.1.0", metadata.RequiredRubyVersion.VersionConstraints[0].Version)
	suite.Equal(">=", metadata.RequiredRubyGemsVersion.VersionConstraints[0].Constraint)
	suite.Equal("0", metadata.RequiredRubyGemsVersion.VersionConstraints[0].Version)
	suite.Equal("3.0.6", metadata.RubygemsVersion)
	suite.Equal("contact@plataformatec.com.br", metadata.Emails[0])
	suite.Equal("contact@plataformatec.com.br", metadata.Email)
	for _, dep := range metadata.Dependencies {
		if dep.Name == "railties" {
			suite.Equal(":runtime", dep.Type)
			suite.Equal(">=", dep.Requirement.VersionConstraints[0].Constraint)
			suite.Equal("4.1.0", dep.Requirement.VersionConstraints[0].Version)
		}
		if dep.Name == "orm_adapter" {
			suite.Equal(":runtime", dep.Type)
			suite.Equal("~>", dep.Requirement.VersionConstraints[0].Constraint)
			suite.Equal("0.1", dep.Requirement.VersionConstraints[0].Version)
		}
		if dep.Name == "thread_safe" {
			suite.Equal(":runtime", dep.Type)
			suite.Equal(">=", dep.Requirement.VersionConstraints[0].Constraint)
			suite.Equal("0.3.4", dep.Requirement.VersionConstraints[0].Version)
		}
		if dep.Name == "bcrypt" {
			suite.Equal(":runtime", dep.Type)
			suite.Equal("~>", dep.Requirement.VersionConstraints[0].Constraint)
			suite.Equal("3.0", dep.Requirement.VersionConstraints[0].Version)
		}
		if dep.Name == "warden" {
			suite.Equal(":runtime", dep.Type)
			suite.Equal("~>", dep.Requirement.VersionConstraints[0].Constraint)
			suite.Equal("1.2.3", dep.Requirement.VersionConstraints[0].Version)
		}
		if dep.Name == "responders" {
			suite.Equal(":runtime", dep.Type)
			suite.Equal(">=", dep.Requirement.VersionConstraints[0].Constraint)
			suite.Equal("0", dep.Requirement.VersionConstraints[0].Version)
		}
	}
	ivars, err := metadata.NumInstanceVars()
	suite.Nil(err)
	suite.Equal(15, ivars)
}

func (suite *SpecTestSuite) TestSpecFromFile() {
	spec, err := FromFile("/tmp", "../../test/fixtures/spec/nokogiri-1.15.3-arm64-darwin.gem")
	suite.Nil(err)
	suite.NotNil(spec)
	suite.Equal("nokogiri", spec.Name)
	suite.Equal("1.15.3", spec.Version)
	suite.Equal("arm64-darwin", spec.OriginalPlatform)
	suite.Equal("< 3.3.dev&>= 2.7", spec.Ruby)
	suite.Equal(">= 0", spec.RubyGems)
	suite.Equal("fa4a027478df9004a2ce91389af7b7b5a4fc790c23492dca43b210a0f8770596", spec.Checksum)
}

func (suite *SpecTestSuite) TestFindIndexOfSpec() {
	specArr := []*Spec{
		{Name: "lol", Version: "1.0.0", OriginalPlatform: "ruby"},
		{Name: "lol2", Version: "1.0.0", OriginalPlatform: "ruby"},
		{Name: "lol3", Version: "1.0.0", OriginalPlatform: "ruby"},
	}
	idx := FindIndexOf(specArr, &Spec{Name: "lol", Version: "1.0.0", OriginalPlatform: "ruby"})
	suite.Equal(0, idx)
	idx = FindIndexOf(specArr, &Spec{Name: "lol", Version: "2.0.0", OriginalPlatform: "ruby"})
	suite.Equal(-1, idx)
	idx = FindIndexOf(specArr, &Spec{Name: "lol3", Version: "1.0.0", OriginalPlatform: "ruby"})
	suite.Equal(2, idx)
	idx = FindIndexOf(specArr, &Spec{Name: "foo", Version: "1.0.0", OriginalPlatform: "ruby"})
	suite.Equal(-1, idx)

}
