package models

import (
	"path/filepath"
	"runtime"
	"testing"

	fixtures "github.com/aquasecurity/bolt-fixtures"
	"github.com/gemfast/server/internal/db"
	"github.com/stretchr/testify/suite"
)

type GemTestSuite struct {
	suite.Suite
	Loader *fixtures.Loader
	DBFile string
}

func (suite *GemTestSuite) SetupTest() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		suite.FailNow("unable to get the current filename")
	}
	dirname := filepath.Dir(filename)
	dbFile := dirname + "/../../test/fixtures/db/test.db"
	fix := dirname + "/../../test/fixtures/db/gems.yaml"
	fixtureFiles := []string{fix}
	l, err := fixtures.New(dbFile, fixtureFiles)
	if err != nil {
		suite.FailNow(err.Error())
	}
	err = l.Load()
	if err != nil {
		suite.FailNow(err.Error())
	}
	suite.Loader = l
	suite.DBFile = dbFile
	db.BoltDB = l.DB()
}

func (suite *GemTestSuite) TearDownTest() {
	suite.Loader.Close()
	// err := os.Remove(suite.DBFile)
	// if err != nil {
	// 	suite.FailNow(err.Error())
	// }
}

func (suite *GemTestSuite) TestSaveGem() {
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

func TestGemTestSuite(t *testing.T) {
	suite.Run(t, new(GemTestSuite))
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
