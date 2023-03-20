package api

import (
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/middleware"
	"github.com/gemfast/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func Run() error {
	router := initRouter()
	port := ":" + config.Env.Port
	log.Info().Str("port", port).Msg("gemfast server ready")
	if config.Env.MirrorEnabled != "false" {
		log.Info().Str("upstream", config.Env.MirrorUpstream).Msg("mirroring upstream gem server")
	}
	return router.Run(port)
}

func initRouter() (r *gin.Engine) {
	// gin.SetMode(gin.ReleaseMode)
	r = gin.Default()
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
	return r
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
	jwtMiddleware, err := middleware.NewJwtMiddleware()
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
	privateTokenAuth.Use(middleware.NewTokenMiddleware())
	{
		configurePrivate(privateTokenAuth)
		privateTokenAuth.POST("/upload", geminaboxUploadGem)
	}
	if config.Env.MirrorEnabled != "false" {
		mirror := r.Group("/")
		configureMirror(mirror)
	}
}

func configureNoneAuth(r *gin.Engine) {
	if config.Env.MirrorEnabled != "false" {
		mirror := r.Group("/")
		configureMirror(mirror)
	}
	private := r.Group("/private")
	configurePrivate(private)
	admin := r.Group("/admin")
	admin.GET("/gems", listGems)
	r.POST("/upload", geminaboxUploadGem)
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
}

func configureAdmin(admin *gin.RouterGroup) {
	admin.GET("/gems", listGems)
	admin.POST("/token", middleware.CreateTokenHandler)
}
