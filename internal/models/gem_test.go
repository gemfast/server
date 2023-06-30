package models

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	fixtures "github.com/aquasecurity/bolt-fixtures"
	"github.com/gemfast/server/internal/db"
	"github.com/stretchr/testify/suite"
)

type ModelsTestSuite struct {
	suite.Suite
	Loader *fixtures.Loader
	DBFile string
}

func (suite *ModelsTestSuite) SetupTest() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		suite.FailNow("unable to get the current filename")
	}
	dirname := filepath.Dir(filename)
	dbFile, _ := ioutil.TempFile("", "ApiTestSuite")
	gemfix := dirname + "/../../test/fixtures/db/gems.yaml"
	userfix := dirname + "/../../test/fixtures/db/users.yaml"
	fixtureFiles := []string{gemfix, userfix}
	l, err := fixtures.New(dbFile.Name(), fixtureFiles)
	if err != nil {
		suite.FailNow(err.Error())
	}
	err = l.Load()
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Loader = l
	suite.DBFile = dbFile.Name()
	db.BoltDB = l.DB()
}

func (suite *ModelsTestSuite) TearDownTest() {
	suite.Loader.Close()
	err := os.Remove(suite.DBFile)
	if err != nil {
		suite.FailNow(err.Error())
	}
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

func (suite *ModelsTestSuite) TestSaveGem() {
	gem := &Gem{
		Name:     "activesupport",
		Number:   "7.0.4.3",
		Platform: "ruby",
		Checksum: "1234567890",
	}
	err := SaveGem(gem)
	suite.Nil(err)
	suite.Equal("activesupport", gem.Name)
	suite.NotNil(gem.InfoChecksum)
	gem.Number = "6.0.4.3"
	ic := gem.InfoChecksum
	err = SaveGem(gem)
	suite.Nil(err)
	suite.Equal("activesupport", gem.Name)
	suite.NotEqual(ic, gem.InfoChecksum)
}

func (suite *ModelsTestSuite) TestGetGems() {
	gems, err := GetGems()
	suite.Nil(err)
	suite.Equal(2, len(gems))
	suite.Equal("chef", gems[0][0].Name)
	suite.Equal("rails", gems[1][0].Name)
}

func (suite *ModelsTestSuite) TestGetGem() {
	gem, err := GetGemVersions("rails")
	suite.Nil(err)
	suite.Equal(1, len(gem))
	suite.Equal("rails", gem[0].Name)
}

func (suite *ModelsTestSuite) TestDeleteGemVersion() {
	count, err := DeleteGemVersion(&Gem{Name: "rails", Number: "6.0.3.rc1"})
	suite.Nil(err)
	suite.Equal(1, count)
}

func (suite *ModelsTestSuite) TestGemAllGemVersions() {
	gemVersions, err := GetAllGemversions()
	suite.Nil(err)
	suite.NotEqual(0, len(gemVersions))
}

func (suite *ModelsTestSuite) TestGemAllGemNames() {
	names := GetAllGemNames()
	suite.Equal(3, len(names))
	suite.Contains(names, "---")
	suite.Contains(names, "chef")
	suite.Contains(names, "rails")
}

func TestGemFromGemParameter(t *testing.T) {
	name := "activesupport-7.0.4.3"
	g := GemFromGemParameter(name)
	if g.Name != "activesupport" {
		t.Errorf("expected gem named activesupport")
	}
	if g.Number != "7.0.4.3" {
		t.Errorf("expected gem version 7.0.4.3")
	}

	name = "activerecord-oracle_enhanced-adapter-1.1.8"
	g = GemFromGemParameter(name)
	if g.Name != "activerecord-oracle_enhanced-adapter" {
		t.Errorf("expected gem named activerecord-oracle_enhanced-adapter")
	}
	if g.Number != "1.1.8" {
		t.Errorf("expected gem version 1.1.8")
	}

	name = ""
	g = GemFromGemParameter(name)
	if g.Name != "" {
		t.Errorf("expected gem name empty")
	}
	if g.Number != "" {
		t.Errorf("expected gem version empty")
	}
}
