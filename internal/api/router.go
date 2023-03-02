package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/models"
	"github.com/gemfast/server/internal/middleware"
	"github.com/rs/zerolog/log"
)

func Run() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	initRouter(r)
	port := ":" + config.Env.Port
	log.Info().Str("port", port).Msg("gemfast server ready")
	if config.Env.Mirror != "" {
		log.Info().Str("upstream", config.Env.MirrorUpstream).Msg("mirroring upstream gem server")
	}
	return r.Run(port)
}

func initRouter(r *gin.Engine) {
	r.Use(gin.Recovery())
	r.HEAD("/", head)
	authMode := config.Env.AuthMode
	log.Info().Str("auth", authMode).Msg("configuring auth strategy")
	switch strings.ToLower(authMode) {
	case "local":
		configureLocalAuth(r)
	case "none":
		configureNoneAuth(r)
	}
}

func configureLocalAuth(r *gin.Engine) {
	err := models.CreateAdminUserIfNotExists()
	if err != nil {
		panic(err)
	}
	err = models.CreateLocalUsers()
	if err != nil {
		panic(err)
	}
	jwtMiddleware, err := initJwtMiddleware()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize auth middleware")
	}
	adminLocalAuth := r.Group("/admin")
	adminLocalAuth.POST("/login", jwtMiddleware.LoginHandler)
	adminLocalAuth.GET("/refresh-token", jwtMiddleware.RefreshHandler)
	adminLocalAuth.Use(jwtMiddleware.MiddlewareFunc())
	{
		configureAdmin(adminLocalAuth)
	}
	privateTokenAuth := r.Group("/private")
	privateTokenAuth.Use(middleware.GinTokenMiddleware())
	{
		configurePrivate(privateTokenAuth)
	}
	if config.Env.Mirror != "" {
		mirror := r.Group("/")
		configureMirror(mirror)
	}
}

func configureNoneAuth(r *gin.Engine) {
	if config.Env.Mirror != "" {
		mirror := r.Group("/")
		configureMirror(mirror)
	}
	private := r.Group("/private")
	configurePrivate(private)
	admin := r.Group("/admin")
	admin.GET("/gems", listGems)
}

func configureMirror(mirror *gin.RouterGroup) {
	mirror.GET("/specs.4.8.gz", mirroredIndexHandler)
	mirror.GET("/latest_specs.4.8.gz", mirroredIndexHandler)
	mirror.GET("/prerelease_specs.4.8.gz", mirroredIndexHandler)
	mirror.GET("/quick/Marshal.4.8/*gemspec.rz", mirroredGemspecRzHandler)
	mirror.GET("/gems/*gem", mirroredGemHandler)
	mirror.GET("/api/v1/dependencies", mirroredDependenciesHandler)
	mirror.GET("/api/v1/dependencies.json", mirroredDependenciesJSONHandler)
	mirror.GET("/info/*gem", mirroredInfoHandler)
	mirror.GET("/versions", mirroredVersionsHandler)
}

func configurePrivate(private *gin.RouterGroup) {
	private.GET("/specs.4.8.gz", localIndexHandler)
	private.GET("/latest_specs.4.8.gz", localIndexHandler)
	private.GET("/prerelease_specs.4.8.gz", localIndexHandler)
	private.GET("/quick/Marshal.4.8/*gemspec.rz", localGemspecRzHandler)
	private.GET("/gems/*gem", localGemHandler)
	private.GET("/api/v1/dependencies", localDependenciesHandler)
	private.GET("/api/v1/dependencies.json", localDependenciesJSONHandler)
	private.POST("/api/v1/gems", localUploadGemHandler)
	private.POST("/upload", geminaboxUploadGem)
}

func configureAdmin(admin *gin.RouterGroup) {
	admin.GET("/gems", listGems)
	admin.POST("/token", createToken)
}