package api

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	fixtures "github.com/aquasecurity/bolt-fixtures"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/license"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

type APITestSuite struct {
	suite.Suite
	Loader      *fixtures.Loader
	DBFile      string
	FixturesDir string
	DB          *bolt.DB
}

func (suite *APITestSuite) SetupTest() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		suite.FailNow("unable to get the current filename")
	}
	dirname := filepath.Dir(filename)
	fixturesDir, err := filepath.Abs(dirname + "/../../test/fixtures")
	if err != nil {
		suite.FailNow(err.Error())
	}
	dbFile, _ := os.CreateTemp("", "APITestSuite")
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
	suite.DB = l.DB()
}

func createTestAPI(testDB *bolt.DB, cfg *config.Config) (*API, error) {
	l, err := license.NewLicense(cfg)
	if err != nil {
		return nil, err
	}
	database := db.NewTestDB(testDB, cfg)
	indexer, err := indexer.NewIndexer(cfg, database)
	if err != nil {
		return nil, err
	}
	f := filter.NewFilter(cfg, l)
	gdb := cve.NewGemAdvisoryDB(cfg, l)
	apiV1Handler := NewAPIV1Handler(cfg, database, indexer, f, gdb)
	rubygemsHandler := NewRubyGemsHandler(cfg, database, indexer, f, gdb)
	api := NewAPI(cfg, database, apiV1Handler, rubygemsHandler)
	api.loadMiddleware()
	api.registerRoutes()
	return api, nil
}

func (suite *APITestSuite) TearDownTest() {
	suite.Loader.Close()
	err := os.Remove(suite.DBFile)
	if err != nil {
		suite.FailNow(err.Error())
	}
}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func (suite *APITestSuite) TestInitRouterNoneAuth() {
	cfg := config.NewConfig()
	cfg.Auth.Type = "none"
	cfg.ACLPath = suite.FixturesDir + "/acl/gemfast_acl.csv"
	cfg.AuthModelPath = suite.FixturesDir + "/acl/auth_model.conf"
	api, err := createTestAPI(suite.DB, cfg)
	suite.Nil(err)
	var paths []string
	for _, route := range api.router.Routes() {
		paths = append(paths, route.Path)
	}
	expectedPaths := []string{
		"/private/api/v1/dependencies",
		"/private/api/v1/dependencies.json",
		"/private/specs.4.8.gz",
		"/private/latest_specs.4.8.gz",
		"/private/prerelease_specs.4.8.gz",
		"/private/quick/Marshal.4.8/:gemspec.rz",
		"/private/gems/:gem",
		"/private/versions",
		"/private/info/:gem",
		"/private/names",
		"/prerelease_specs.4.8.gz",
		"/admin/api/v1/gems",
		"/admin/api/v1/gems/:gem",
		"/admin/api/v1/users",
		"/admin/api/v1/users/:username",
		"/api/v1/dependencies",
		"/api/v1/dependencies.json",
		"/admin/api/v1/auth",
		"/up",
		"/specs.4.8.gz",
		"/latest_specs.4.8.gz",
		"/quick/Marshal.4.8/:gemspec.rz",
		"/gems/:gem",
		"/info/*gem",
		"/versions",
		"/admin/api/v1/token",
		"/private/api/v1/gems",
		"/private/upload",
		"/admin/api/v1/users/:username",
		"/private/api/v1/gems/yank",
		"/admin/api/v1/users/:username/role/:role",
	}
	for _, p := range expectedPaths {
		suite.Contains(paths, p)
	}
	suite.NotContains(paths, "/admin/api/v1/login")
}

func (suite *APITestSuite) TestInitRouterLocalAuth() {
	cfg := config.NewConfig()
	cfg.Auth.Type = "local"
	cfg.ACLPath = suite.FixturesDir + "/acl/gemfast_acl.csv"
	cfg.AuthModelPath = suite.FixturesDir + "/acl/auth_model.conf"
	api, err := createTestAPI(suite.DB, cfg)
	suite.Nil(err)
	var paths []string
	for _, route := range api.router.Routes() {
		paths = append(paths, route.Path)
	}
	expectedPaths := []string{
		"/private/api/v1/dependencies",
		"/private/api/v1/dependencies.json",
		"/private/specs.4.8.gz",
		"/private/latest_specs.4.8.gz",
		"/private/prerelease_specs.4.8.gz",
		"/private/quick/Marshal.4.8/:gemspec.rz",
		"/private/gems/:gem",
		"/private/versions",
		"/private/info/:gem",
		"/private/names",
		"/prerelease_specs.4.8.gz",
		"/admin/api/v1/gems",
		"/admin/api/v1/gems/:gem",
		"/admin/api/v1/users",
		"/admin/api/v1/users/:username",
		"/api/v1/dependencies",
		"/api/v1/dependencies.json",
		"/admin/api/v1/auth",
		"/up",
		"/specs.4.8.gz",
		"/latest_specs.4.8.gz",
		"/quick/Marshal.4.8/:gemspec.rz",
		"/gems/:gem",
		"/info/*gem",
		"/versions",
		"/private/api/v1/gems",
		"/private/upload",
		"/admin/api/v1/token",
		"/private/api/v1/gems/yank",
		"/admin/api/v1/users/:username",
		"/admin/api/v1/users/:username/role/:role",
	}
	for _, p := range expectedPaths {
		suite.Contains(paths, p)
	}
	suite.NotContains(paths, "/github/callback")
}

func (suite *APITestSuite) TestInitRouterGitHubAuth() {
	cfg := config.NewConfig()
	cfg.Auth.Type = "github"
	cfg.ACLPath = suite.FixturesDir + "/acl/gemfast_acl.csv"
	cfg.AuthModelPath = suite.FixturesDir + "/acl/auth_model.conf"
	api, err := createTestAPI(suite.DB, cfg)
	suite.Nil(err)
	var paths []string
	for _, route := range api.router.Routes() {
		paths = append(paths, route.Path)
	}
	expectedPaths := []string{
		"/private/api/v1/dependencies",
		"/private/api/v1/dependencies.json",
		"/private/specs.4.8.gz",
		"/private/latest_specs.4.8.gz",
		"/private/prerelease_specs.4.8.gz",
		"/private/quick/Marshal.4.8/:gemspec.rz",
		"/private/gems/:gem",
		"/private/versions",
		"/private/info/:gem",
		"/private/names",
		"/prerelease_specs.4.8.gz",
		"/admin/api/v1/gems",
		"/admin/api/v1/gems/:gem",
		"/admin/api/v1/users",
		"/admin/api/v1/users/:username",
		"/api/v1/dependencies",
		"/api/v1/dependencies.json",
		"/admin/api/v1/auth",
		"/up",
		"/specs.4.8.gz",
		"/latest_specs.4.8.gz",
		"/quick/Marshal.4.8/:gemspec.rz",
		"/gems/:gem",
		"/info/*gem",
		"/versions",
		"/private/api/v1/gems",
		"/private/upload",
		"/admin/api/v1/token",
		"/private/api/v1/gems/yank",
		"/admin/api/v1/users/:username",
		"/admin/api/v1/users/:username/role/:role",
		"/github/callback",
	}
	for _, p := range expectedPaths {
		suite.Contains(paths, p)
	}
}
