package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
}

func (suite *ConfigTestSuite) TestReadJWTSecretKeyFromPath() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		suite.FailNow("unable to get the current filename")
	}
	dirname := filepath.Dir(filename)
	fixturesDir, err := filepath.Abs(dirname + "/../../test/fixtures")
	if err != nil {
		suite.FailNow(err.Error())
	}
	jwt := readJWTSecretKeyFromPath(fixturesDir + "/jwt/.jwt_secret_key")
	defer os.Remove(fixturesDir + "/jwt/.jwt_secret_key")
	suite.NotEmpty(jwt)
	suite.NotEqual("test", jwt)
	jwt = readJWTSecretKeyFromPath(fixturesDir + "/jwt/.jwt_test")
	suite.Equal("test", jwt)
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
