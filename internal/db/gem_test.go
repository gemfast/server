package db

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	fixtures "github.com/aquasecurity/bolt-fixtures"
	"github.com/gemfast/server/internal/config"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

type ModelsTestSuite struct {
	suite.Suite
	Loader *fixtures.Loader
	DBFile string
	db     *bolt.DB
}

func (suite *ModelsTestSuite) SetupTest() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		suite.FailNow("unable to get the current filename")
	}
	dirname := filepath.Dir(filename)
	dbFile, _ := os.CreateTemp("", "ModelsTestSuite")
	fixturesDir, err := filepath.Abs(dirname + "/../../test/fixtures")
	if err != nil {
		suite.FailNow(err.Error())
	}
	gemfix := fixturesDir + "/db/gems.yaml"
	userfix := fixturesDir + "/db/users.yaml"
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
	suite.db = l.DB()
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

func (suite *ModelsTestSuite) TestSaveGemPrivate() {
	gem := &Gem{
		Name:     "activesupport",
		Number:   "7.0.4.3",
		Platform: "ruby",
		Checksum: "1234567890",
	}
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	err := db.SaveGem("private", gem)
	suite.Nil(err)
	suite.Equal("activesupport", gem.Name)
	suite.NotNil(gem.InfoChecksum)
	gem.Number = "6.0.4.3"
	ic := gem.InfoChecksum
	err = db.SaveGem("private", gem)
	suite.Nil(err)
	suite.Equal("activesupport", gem.Name)
	suite.NotEqual(ic, gem.InfoChecksum)
}

func (suite *ModelsTestSuite) TestSaveGemMirror() {
	gem := &Gem{
		Name:     "activesupport",
		Number:   "7.0.4.3",
		Platform: "ruby",
		Checksum: "1234567890",
	}
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	err := db.SaveGem("rubygems.org", gem)
	suite.Nil(err)
	suite.Equal("activesupport", gem.Name)
	suite.NotNil(gem.InfoChecksum)
	gem.Number = "6.0.4.3"
	ic := gem.InfoChecksum
	err = db.SaveGem("rubygems.org", gem)
	suite.Nil(err)
	suite.Equal("activesupport", gem.Name)
	suite.NotEqual(ic, gem.InfoChecksum)
}

func (suite *ModelsTestSuite) TestGetGems() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	gems, err := db.GetGems("private")
	suite.Nil(err)
	suite.Equal(2, len(gems))
	suite.Equal("chef", gems[0][0].Name)
	suite.Equal("rails", gems[1][0].Name)
	gems, err = db.GetGems("rubygems.org")
	suite.Nil(err)
	suite.Equal(1, len(gems))
	suite.Equal("good_job", gems[0][0].Name)
}

func (suite *ModelsTestSuite) TestGetGem() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	gem, err := db.GetGemVersions("private", "rails")
	suite.Nil(err)
	suite.Equal(1, len(gem))
	suite.Equal("rails", gem[0].Name)
	gem, err = db.GetGemVersions("rubygems.org", "good_job")
	suite.Nil(err)
	suite.Equal(1, len(gem))
	suite.Equal("good_job", gem[0].Name)
}

func (suite *ModelsTestSuite) TestDeleteGemVersion() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	count, err := db.DeleteGemVersion("private", &Gem{Name: "rails", Number: "6.0.3.rc1"})
	suite.Nil(err)
	suite.Equal(1, count)
	count, err = db.DeleteGemVersion("rubygems.org", &Gem{Name: "good_job", Number: "3.7.14"})
	suite.Nil(err)
	suite.Equal(1, count)
}

func (suite *ModelsTestSuite) TestGemAllGemVersions() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	gemVersions, err := db.GetAllGemversions("private")
	suite.Nil(err)
	suite.NotEqual(0, len(gemVersions))
	gemVersions, err = db.GetAllGemversions("rubygems.org")
	suite.Nil(err)
	suite.NotEqual(0, len(gemVersions))
}

func (suite *ModelsTestSuite) TestGemAllGemNames() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	names := db.GetAllGemNames("private")
	suite.Equal(3, len(names))
	suite.Contains(names, "---")
	suite.Contains(names, "chef")
	suite.Contains(names, "rails")

	names = db.GetAllGemNames("rubygems.org")
	suite.Equal(2, len(names))
	suite.Contains(names, "---")
	suite.Contains(names, "good_job")
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
