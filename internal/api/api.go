package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gemfast/server/internal/acl"
	"github.com/gemfast/server/internal/api/handlers"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/middleware"
	"github.com/gemfast/server/internal/ui"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

const adminAPIPath = "/admin/api/v1"

type API struct {
	apiV1Handler          *handlers.APIV1Handler
	rubygemsHandler       *handlers.RubyGemsHandler
	rubygemsMirrorHandler *handlers.RubyGemsMirrorHandler
	router                *gin.Engine
	cfg                   *config.Config
	db                    *db.DB
	tokenMiddleware       *middleware.TokenMiddleware
	githubMiddleware      *middleware.GitHubMiddleware
	jwtMiddleware         *middleware.JWTMiddleware
}

func NewAPI(cfg *config.Config, db *db.DB, indexer *indexer.Indexer, filter *filter.RegexFilter, advisoryDB *cve.GemAdvisoryDB) *API {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	aclInstance := acl.NewACL(cfg, db)
	apiV1Handler := handlers.NewAPIV1Handler(cfg, db, indexer, filter, advisoryDB, aclInstance)
	rubygemsHandler := handlers.NewRubyGemsHandler(cfg, db, indexer, filter, advisoryDB)
	rubygemsMirrorHandler := handlers.NewRubyGemsMirrorHandler(cfg, db, indexer, filter, advisoryDB)
	return &API{
		apiV1Handler:          apiV1Handler,
		rubygemsHandler:       rubygemsHandler,
		rubygemsMirrorHandler: rubygemsMirrorHandler,
		router:                router,
		cfg:                   cfg,
		db:                    db,
	}
}

func (api *API) Run() {
	api.loadMiddleware()
	api.registerRoutes()
	port := fmt.Sprintf(":%d", api.cfg.Port)
	if api.cfg.Mirrors[0].Enabled {
		log.Info().Str("detail", api.cfg.Mirrors[0].Upstream).Msg("mirroring upstream gem server")
	}
	log.Info().Str("detail", fmt.Sprintf("http://0.0.0.0%s", port)).Msg("gemfast server started")
	err := api.router.Run(port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}
}

func (api *API) loadMiddleware() {
	acl := acl.NewACL(api.cfg, api.db)
	api.tokenMiddleware = middleware.NewTokenMiddleware(acl, api.db)
	api.githubMiddleware = middleware.NewGitHubMiddleware(api.cfg, acl, api.db)
	api.jwtMiddleware = middleware.NewJWTMiddleware(api.cfg, acl, api.db)
	store := cookie.NewStore([]byte("secret"))
	api.router.Use(sessions.Sessions("gemfast", store))
	if !api.cfg.MetricsDisabled {
		p := ginprometheus.NewPrometheus("gemfast")
		p.Use(api.router)
	}
}

func (api *API) registerRoutes() {
	api.router.Use(gin.Recovery())
	ui := ui.NewUI(api.cfg, api.db)
	api.router.SetHTMLTemplate(ui.Templates)
	api.router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui")
	})
	api.router.GET("/up", api.apiV1Handler.Health)
	authMode := api.cfg.Auth.Type
	log.Info().Str("detail", authMode).Msg("configuring auth strategy")
	switch strings.ToLower(authMode) {
	case "github":
		api.configureGitHubAuth(ui)
	case "local":
		api.configureLocalAuth()
	case "none":
		api.configureNoneAuth(ui)
	default:
		log.Fatal().Msg(fmt.Sprintf("invalid auth type: %s", authMode))
	}
}

func (api *API) configureGitHubAuth(ui *ui.UI) {
	adminGitHubAuth := api.router.Group(adminAPIPath)
	adminGitHubAuth.POST("/login", api.githubMiddleware.GitHubLoginHandler)
	slash := api.router.Group("/")
	slash.GET("/github/callback", api.githubMiddleware.GitHubCallbackHandler)
	adminGitHubAuth.Use(api.githubMiddleware.GitHubMiddlewareFunc())
	{
		api.configureAdmin(adminGitHubAuth)
	}
	if !api.cfg.UIDisabled {
		api.router.StaticFS("/ui/assets", http.FS(ui.Assets))
		uiGroup := api.router.Group("/ui")
		uiGroup.Use(api.githubMiddleware.GitHubMiddlewareFunc())
		{
			api.configureUI(ui, uiGroup)
		}
		api.router.GET("/ui/github/logout", api.githubMiddleware.GitHubLogoutHandler)
		log.Info().Str("detail", "/ui").Msg("gemfast ui enabled")
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

func (api *API) configureNoneAuth(ui *ui.UI) {
	if api.cfg.Mirrors[0].Enabled {
		mirror := api.router.Group("/")
		api.configureMirror(mirror)
	}
	private := api.router.Group(filepath.Join("/", api.cfg.PrivateGemsNamespace))
	api.configurePrivateRead(private)
	api.configurePrivateWrite(private)
	admin := api.router.Group(adminAPIPath)
	api.configureAdmin(admin)
	if !api.cfg.UIDisabled {
		api.router.StaticFS("/ui/assets", http.FS(ui.Assets))
		uiGroup := api.router.Group("/ui")
		api.configureUI(ui, uiGroup)
		log.Info().Str("detail", "/ui").Msg("gemfast ui enabled")
	}
}

// /
func (api *API) configureMirror(mirror *gin.RouterGroup) {
	mirror.GET("/specs.4.8.gz", api.rubygemsMirrorHandler.GetIndex)
	mirror.GET("/latest_specs.4.8.gz", api.rubygemsMirrorHandler.GetIndex)
	mirror.GET("/prerelease_specs.4.8.gz", api.rubygemsMirrorHandler.GetIndex)
	mirror.GET("/quick/Marshal.4.8/:gemspec.rz", api.rubygemsMirrorHandler.GetGemspecRz)
	mirror.GET("/gems/:gem", api.rubygemsMirrorHandler.GetGem)
	mirror.GET("/api/v1/dependencies", api.rubygemsMirrorHandler.GetGemDependencies)
	mirror.GET("/api/v1/dependencies.json", api.rubygemsMirrorHandler.GetGemDependenciesJSON)
	mirror.GET("/info/*gem", api.rubygemsMirrorHandler.GetGemInfo)
	mirror.GET("/versions", api.rubygemsMirrorHandler.GetGemVersionsCompact)
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
	private.GET("/specs.4.8.gz", api.rubygemsHandler.GetIndex)
	private.GET("/latest_specs.4.8.gz", api.rubygemsHandler.GetIndex)
	private.GET("/prerelease_specs.4.8.gz", api.rubygemsHandler.GetIndex)
	private.GET("/quick/Marshal.4.8/:gemspec.rz", api.rubygemsHandler.GetGemspecRz)
	private.GET("/gems/:gem", api.rubygemsHandler.GetGem)
	private.GET("/api/v1/dependencies", api.rubygemsHandler.GetGemDependencies)
	private.GET("/api/v1/dependencies.json", api.rubygemsHandler.GetGemDependenciesJSON)
	private.GET("/versions", api.rubygemsHandler.GetGemVersionsCompact)
	private.GET("/info/:gem", api.rubygemsHandler.GetGemInfo)
	private.GET("/names", api.rubygemsHandler.GetGemNames)
}

// /private
func (api *API) configurePrivateWrite(private *gin.RouterGroup) {
	private.POST("/api/v1/gems", api.rubygemsHandler.UploadGem)
	private.DELETE("/api/v1/gems/yank", api.rubygemsHandler.YankGem)
	private.POST("/upload", api.rubygemsHandler.GeminaboxUploadGem)
}

// /admin
func (api *API) configureAdmin(admin *gin.RouterGroup) {

	admin.GET("/gems/:source", api.apiV1Handler.ListGems)
	admin.GET("/gems/:source/:gem", api.apiV1Handler.GetGem)
	admin.GET("/gems/:source/search/:name", api.apiV1Handler.SearchGems)
	admin.GET("/gems/:source/prefix/:prefix", api.apiV1Handler.PrefixScanGems)

	// Auth and ACL management endpoints
	if api.cfg.Auth.Type != "none" {
		admin.GET("/users", api.apiV1Handler.ListUsers)
		admin.GET("/users/:username", api.apiV1Handler.GetUser)
		if api.cfg.Auth.Type == "local" {
			admin.POST("/users", api.apiV1Handler.CreateUser)
			admin.PUT("/users/:username/password", api.apiV1Handler.UpdateUserPassword)
		}
		admin.DELETE("/users/:username", api.apiV1Handler.DeleteUser)
		admin.PUT("/users/:username/role", api.apiV1Handler.UpdateUserRole)
		admin.GET("/auth", api.apiV1Handler.GetAuthMode)
		admin.POST("/token", api.tokenMiddleware.CreateUserTokenHandler)
		admin.GET("/acl/policies", api.apiV1Handler.ListPolicies)
		admin.POST("/acl/policies", api.apiV1Handler.AddPolicy)
		admin.DELETE("/acl/policies", api.apiV1Handler.RemovePolicy)
	}
	// Database management endpoints
	admin.GET("/backup", api.apiV1Handler.Backup)
	admin.GET("/stats/db", api.apiV1Handler.DBStats)
	admin.GET("/stats/bucket", api.apiV1Handler.BucketStats)
	admin.POST("/webhooks", api.apiV1Handler.CreateWebhook)
	admin.GET("/webhooks", api.apiV1Handler.ListWebhooks)
	admin.GET("/webhooks/:id", api.apiV1Handler.GetWebhook)
	admin.PUT("/webhooks/:id", api.apiV1Handler.UpdateWebhook)
	admin.DELETE("/webhooks/:id", api.apiV1Handler.DeleteWebhook)
	admin.GET("/webhooks/:id/history", api.apiV1Handler.WebhookHistory)
}

// /ui
func (api *API) configureUI(ui *ui.UI, uiPath *gin.RouterGroup) {
	uiPath.GET("/", ui.Index)
	uiPath.GET("/gems", ui.Gems)
	uiPath.GET("/upload", ui.UploadGem)
	uiPath.POST("/upload", api.rubygemsHandler.GeminaboxUploadGem)
	uiPath.GET("/download/:gem", api.rubygemsHandler.GetGem)
	uiPath.GET("/tokens", ui.AccessTokens)
	uiPath.POST("/gems/search", ui.SearchGems)
	uiPath.GET("/gems/:source/prefix", ui.GemsByPrefix)
	uiPath.GET("/gems/:source/prefix/:prefix", ui.GemsData)
	uiPath.GET("/gems/:source/prefix/:prefix/inspect/:gem", ui.GemsInspect)
}
