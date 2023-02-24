package api

import (
	"fmt"
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
	r.POST("/login", jwtMiddleware.LoginHandler)
	localAuth := r.Group("/")
	localAuth.GET("/refresh-token", jwtMiddleware.RefreshHandler)
	localAuth.Use(jwtMiddleware.MiddlewareFunc())
	{
		localAuth.POST("token", createToken)
	}
	tokenAuth := r.Group("/")
	tokenAuth.Use(middleware.GinTokenMiddleware())
	{
		tokenAuth.StaticFile("/specs.4.8.gz", fmt.Sprintf("%s/specs.4.8.gz", config.Env.Dir))
		tokenAuth.StaticFile("/latest_specs.4.8.gz", fmt.Sprintf("%s/latest_specs.4.8.gz", config.Env.Dir))
		tokenAuth.StaticFile("/prerelease_specs.4.8.gz", fmt.Sprintf("%s/prerelease_specs.4.8.gz", config.Env.Dir))
		tokenAuth.GET("/quick/Marshal.4.8/*gemspec.rz", getGemspecRz)
		tokenAuth.GET("/gems/*gem", getGem)
		tokenAuth.GET("/api/v1/dependencies", getDependencies)
		tokenAuth.GET("/api/v1/dependencies.json", getDependenciesJSON)
		tokenAuth.POST("/api/v1/gems", uploadGem)
		tokenAuth.POST("/upload", geminaboxUploadGem)
	}
}

func configureNoneAuth(r *gin.Engine) {
	r.POST("create_token", createToken)
	r.StaticFile("/specs.4.8.gz", fmt.Sprintf("%s/specs.4.8.gz", config.Env.Dir))
	r.StaticFile("/latest_specs.4.8.gz", fmt.Sprintf("%s/latest_specs.4.8.gz", config.Env.Dir))
	r.StaticFile("/prerelease_specs.4.8.gz", fmt.Sprintf("%s/prerelease_specs.4.8.gz", config.Env.Dir))
	r.GET("/quick/Marshal.4.8/*gemspec.rz", getGemspecRz)
	r.GET("/gems/*gem", getGem)
	r.GET("/api/v1/dependencies", getDependencies)
	r.GET("/api/v1/dependencies.json", getDependenciesJSON)
	r.POST("/api/v1/gems", uploadGem)
	r.POST("/upload", geminaboxUploadGem)
	r.GET("/gems", listGems)
}
