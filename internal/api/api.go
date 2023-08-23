package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/middleware"
	"github.com/gemfast/server/internal/ui"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const adminAPIPath = "/admin/api/v1"

type API struct {
	apiV1Handler     *APIV1Handler
	rubygemsHandler  *RubyGemsHandler
	router           *gin.Engine
	cfg              *config.Config
	db               *db.DB
	tokenMiddleware  *middleware.TokenMiddleware
	githubMiddleware *middleware.GitHubMiddleware
	jwtMiddleware    *middleware.JWTMiddleware
}

func NewAPI(cfg *config.Config, db *db.DB, apiV1Handler *APIV1Handler, rubygemsHandler *RubyGemsHandler) *API {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	return &API{
		apiV1Handler:    apiV1Handler,
		rubygemsHandler: rubygemsHandler,
		router:          router,
		cfg:             cfg,
		db:              db,
	}
}

func (api *API) Run() {
	api.loadMiddleware()
	api.registerRoutes()
	port := fmt.Sprintf(":%d", api.cfg.Port)
	if api.cfg.Mirrors[0].Enabled {
		log.Info().Str("detail", api.cfg.Mirrors[0].Upstream).Msg("mirroring upstream gem server")
	}
	log.Info().Str("detail", port).Msg("gemfast server listening on port")
	err := api.router.Run(port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}
}

func (api *API) loadMiddleware() {
	acl := middleware.NewACL(api.cfg)
	api.tokenMiddleware = middleware.NewTokenMiddleware(acl, api.db)
	api.githubMiddleware = middleware.NewGitHubMiddleware(api.cfg, acl, api.db)
	api.jwtMiddleware = middleware.NewJWTMiddleware(api.cfg, acl, api.db)
}

func (api *API) registerRoutes() {
	ui := ui.NewUI(api.cfg, api.db)
	if !api.cfg.UIDisabled {
		api.router.StaticFS("/ui/assets", http.FS(ui.Assets))
		api.router.SetHTMLTemplate(ui.Templates)
		log.Info().Str("detail", "/ui").Msg("gemfast ui available enabled")
	}
	api.router.Use(gin.Recovery())
	api.router.GET("/up", api.apiV1Handler.health)
	api.configureUI(ui, api.router.Group("/ui"))
	authMode := api.cfg.Auth.Type
	log.Info().Str("detail", authMode).Msg("configuring auth strategy")
	switch strings.ToLower(authMode) {
	case "github":
		api.configureGitHubAuth()
	case "local":
		api.configureLocalAuth()
	case "none":
		api.configureNoneAuth()
	default:
		log.Fatal().Msg(fmt.Sprintf("invalid auth type: %s", authMode))
	}
}

func (api *API) configureGitHubAuth() {
	adminGitHubAuth := api.router.Group(adminAPIPath)
	adminGitHubAuth.POST("/login", api.githubMiddleware.GitHubLoginHandler)
	slash := api.router.Group("/")
	slash.GET("/github/callback", api.githubMiddleware.GitHubCallbackHandler)
	adminGitHubAuth.Use(api.githubMiddleware.GitHubMiddlewareFunc())
	{
		api.configureAdmin(adminGitHubAuth)
	}
	api.configurePrivate()
}

func (api *API) configureLocalAuth() {
	err := api.db.CreateAdminUserIfNotExists()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create admin user")
	}
	err = api.db.CreateLocalUsers()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create local users")
	}
	jwtMiddleware, err := api.jwtMiddleware.InitJwtMiddleware()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize auth middleware")
	}
	adminLocalAuth := api.router.Group(adminAPIPath)
	adminLocalAuth.POST("/login", jwtMiddleware.LoginHandler)
	adminLocalAuth.GET("/refresh-token", jwtMiddleware.RefreshHandler)
	adminLocalAuth.Use(jwtMiddleware.MiddlewareFunc())
	{
		api.configureAdmin(adminLocalAuth)
	}
	api.configurePrivate()
}

func (api *API) configureNoneAuth() {
	if api.cfg.Mirrors[0].Enabled {
		mirror := api.router.Group("/")
		api.configureMirror(mirror)
	}
	private := api.router.Group(filepath.Join("/", api.cfg.PrivateGemsNamespace))
	api.configurePrivateRead(private)
	api.configurePrivateWrite(private)
	admin := api.router.Group(adminAPIPath)
	api.configureAdmin(admin)
}

// /
func (api *API) configureMirror(mirror *gin.RouterGroup) {
	mirror.GET("/specs.4.8.gz", api.rubygemsHandler.mirroredIndexHandler)
	mirror.GET("/latest_specs.4.8.gz", api.rubygemsHandler.mirroredIndexHandler)
	mirror.GET("/prerelease_specs.4.8.gz", api.rubygemsHandler.mirroredIndexHandler)
	mirror.GET("/quick/Marshal.4.8/:gemspec.rz", api.rubygemsHandler.mirroredGemspecRzHandler)
	mirror.GET("/gems/:gem", api.rubygemsHandler.mirroredGemHandler)
	mirror.GET("/api/v1/dependencies", api.rubygemsHandler.mirroredDependenciesHandler)
	mirror.GET("/api/v1/dependencies.json", api.rubygemsHandler.mirroredDependenciesJSONHandler)
	mirror.GET("/info/*gem", api.rubygemsHandler.mirroredInfoHandler)
	mirror.GET("/versions", api.rubygemsHandler.mirroredVersionsHandler)
}

// /private
func (api *API) configurePrivate() {
	privateTokenAuth := api.router.Group(filepath.Join("/", api.cfg.PrivateGemsNamespace))
	privateTokenAuth.Use(api.tokenMiddleware.TokenMiddlewareFunc())
	{
		if !api.cfg.Auth.AllowAnonymousRead {
			api.configurePrivateRead(privateTokenAuth)
		}
		api.configurePrivateWrite(privateTokenAuth)
	}
	if api.cfg.Mirrors[0].Enabled {
		mirror := api.router.Group("/")
		api.configureMirror(mirror)
	}
	if api.cfg.Auth.AllowAnonymousRead {
		private := api.router.Group(filepath.Join("/", api.cfg.PrivateGemsNamespace))
		api.configurePrivateRead(private)
	}
}

// /private
func (api *API) configurePrivateRead(private *gin.RouterGroup) {
	private.GET("/specs.4.8.gz", api.rubygemsHandler.localIndexHandler)
	private.GET("/latest_specs.4.8.gz", api.rubygemsHandler.localIndexHandler)
	private.GET("/prerelease_specs.4.8.gz", api.rubygemsHandler.localIndexHandler)
	private.GET("/quick/Marshal.4.8/:gemspec.rz", api.rubygemsHandler.localGemspecRzHandler)
	private.GET("/gems/:gem", api.rubygemsHandler.localGemHandler)
	private.GET("/api/v1/dependencies", api.rubygemsHandler.localDependenciesHandler)
	private.GET("/api/v1/dependencies.json", api.rubygemsHandler.localDependenciesJSONHandler)
	private.GET("/versions", api.rubygemsHandler.localVersionsHandler)
	private.GET("/info/:gem", api.rubygemsHandler.localInfoHandler)
	private.GET("/names", api.rubygemsHandler.localNamesHandler)
}

// /private
func (api *API) configurePrivateWrite(private *gin.RouterGroup) {
	private.POST("/api/v1/gems", api.rubygemsHandler.localUploadGemHandler)
	private.DELETE("/api/v1/gems/yank", api.rubygemsHandler.localYankHandler)
	private.POST("/upload", api.rubygemsHandler.geminaboxUploadGem)
}

// /admin
func (api *API) configureAdmin(admin *gin.RouterGroup) {
	admin.GET("/auth", api.apiV1Handler.authMode)
	admin.POST("/token", api.tokenMiddleware.CreateUserTokenHandler)
	admin.GET("/gems", api.apiV1Handler.listGems)
	admin.GET("/gems/:gem", api.apiV1Handler.getGem)
	admin.GET("/gems/search/:name", api.apiV1Handler.searchGems)
	admin.GET("/gems/prefix/:prefix", api.apiV1Handler.prefixScanGems)
	admin.GET("/users", api.apiV1Handler.listUsers)
	admin.GET("/users/:username", api.apiV1Handler.getUser)
	admin.DELETE("/users/:username", api.apiV1Handler.deleteUser)
	admin.PUT("/users/:username/role/:role", api.apiV1Handler.setUserRole)
	admin.GET("/backup", api.apiV1Handler.backup)
	admin.GET("/stats/db", api.apiV1Handler.dbStats)
	admin.GET("/stats/bucket", api.apiV1Handler.bucketStats)
}

// /ui
func (api *API) configureUI(ui *ui.UI, uiPath *gin.RouterGroup) {
	uiPath.GET("/", ui.Index)
	uiPath.GET("/gems", ui.Gems)
	uiPath.GET("/upload", ui.UploadGem)
	uiPath.GET("/license", ui.License)
	uiPath.POST("/gems/search", ui.SearchGems)
	uiPath.GET("/gems/:source/prefix", ui.GemsByPrefix)
	uiPath.GET("/gems/:source/prefix/:prefix", ui.GemsData)
	uiPath.GET("/gems/:source/prefix/:prefix/inspect/:gem", ui.GemsInspect)
}
