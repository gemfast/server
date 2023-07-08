package api

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	fixtures "github.com/aquasecurity/bolt-fixtures"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/stretchr/testify/suite"
)

type ApiTestSuite struct {
	suite.Suite
	Loader      *fixtures.Loader
	DBFile      string
	FixturesDir string
}

func (suite *ApiTestSuite) SetupTest() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		suite.FailNow("unable to get the current filename")
	}
	dirname := filepath.Dir(filename)
	fixturesDir, err := filepath.Abs(dirname + "/../../test/fixtures")
	if err != nil {
		suite.FailNow(err.Error())
	}
	dbFile, _ := os.CreateTemp("", "ApiTestSuite")
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
	suite.FixturesDir = fixturesDir
	db.BoltDB = l.DB()
}

func (suite *ApiTestSuite) TearDownTest() {
	suite.Loader.Close()
	err := os.Remove(suite.DBFile)
	if err != nil {
		suite.FailNow(err.Error())
	}
}

func TestApiTestSuite(t *testing.T) {
	suite.Run(t, new(ApiTestSuite))
}

func (suite *ApiTestSuite) TestInitRouterNoneAuth() {
	config.LoadConfig()
	config.Cfg.Auth.Type = "none"
	config.Cfg.ACLPath = suite.FixturesDir + "/gemfast_acl.csv"
	config.Cfg.AuthModelPath = suite.FixturesDir + "/auth_model.conf"
	r := initRouter()
	var paths []string
	for _, route := range r.Routes() {
		paths = append(paths, route.Path)
	}
	expectedPaths := []string{"/private/api/v1/dependencies", "/private/api/v1/dependencies.json", "/private/specs.4.8.gz", "/private/latest_specs.4.8.gz", "/private/prerelease_specs.4.8.gz", "/private/quick/Marshal.4.8/:gemspec.rz", "/private/gems/:gem", "/private/versions", "/private/info/:gem", "/private/names", "/prerelease_specs.4.8.gz", "/admin/gems", "/admin/gems/:gem", "/admin/users", "/admin/users/:username", "/api/v1/dependencies", "/api/v1/dependencies.json", "/auth", "/up", "/specs.4.8.gz", "/latest_specs.4.8.gz", "/quick/Marshal.4.8/:gemspec.rz", "/gems/:gem", "/info/*gem", "/versions", "/admin/token", "/private/api/v1/gems", "/private/upload", "/admin/users/:username", "/private/api/v1/gems/yank", "/admin/users/:username/role/:role"}
	for _, p := range expectedPaths {
		suite.Contains(paths, p)
	}
	suite.NotContains(paths, "/admin/login")
}

func (suite *ApiTestSuite) TestInitRouterLocalAuth() {
	config.LoadConfig()
	config.Cfg.Auth.Type = "local"
	config.Cfg.ACLPath = suite.FixturesDir + "/acl/gemfast_acl.csv"
	config.Cfg.AuthModelPath = suite.FixturesDir + "/acl/auth_model.conf"
	r := initRouter()
	var paths []string
	for _, route := range r.Routes() {
		paths = append(paths, route.Path)
	}
	expectedPaths := []string{"/private/api/v1/dependencies", "/private/api/v1/dependencies.json", "/private/specs.4.8.gz", "/private/latest_specs.4.8.gz", "/private/prerelease_specs.4.8.gz", "/private/quick/Marshal.4.8/:gemspec.rz", "/private/gems/:gem", "/private/versions", "/private/info/:gem", "/private/names", "/prerelease_specs.4.8.gz", "/admin/gems", "/admin/gems/:gem", "/admin/users", "/admin/users/:username", "/api/v1/dependencies", "/api/v1/dependencies.json", "/auth", "/up", "/specs.4.8.gz", "/latest_specs.4.8.gz", "/quick/Marshal.4.8/:gemspec.rz", "/gems/:gem", "/info/*gem", "/versions", "/private/api/v1/gems", "/private/upload", "/admin/token", "/private/api/v1/gems/yank", "/admin/users/:username", "/admin/users/:username/role/:role"}
	for _, p := range expectedPaths {
		suite.Contains(paths, p)
	}
	suite.NotContains(paths, "/github/callback")
}

func (suite *ApiTestSuite) TestInitRouterGitHubAuth() {
	config.LoadConfig()
	config.Cfg.Auth.Type = "github"
	config.Cfg.ACLPath = suite.FixturesDir + "/acl/gemfast_acl.csv"
	config.Cfg.AuthModelPath = suite.FixturesDir + "/acl/auth_model.conf"
	r := initRouter()
	var paths []string
	for _, route := range r.Routes() {
		paths = append(paths, route.Path)
	}
	expectedPaths := []string{"/private/api/v1/dependencies", "/private/api/v1/dependencies.json", "/private/specs.4.8.gz", "/private/latest_specs.4.8.gz", "/private/prerelease_specs.4.8.gz", "/private/quick/Marshal.4.8/:gemspec.rz", "/private/gems/:gem", "/private/versions", "/private/info/:gem", "/private/names", "/prerelease_specs.4.8.gz", "/admin/gems", "/admin/gems/:gem", "/admin/users", "/admin/users/:username", "/api/v1/dependencies", "/api/v1/dependencies.json", "/auth", "/up", "/specs.4.8.gz", "/latest_specs.4.8.gz", "/quick/Marshal.4.8/:gemspec.rz", "/gems/:gem", "/info/*gem", "/versions", "/private/api/v1/gems", "/private/upload", "/admin/token", "/private/api/v1/gems/yank", "/admin/users/:username", "/admin/users/:username/role/:role", "/github/callback"}
	for _, p := range expectedPaths {
		suite.Contains(paths, p)
	}
}
