package api

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gscho/gemfast/internal/config"
	"github.com/gscho/gemfast/internal/models"
	"github.com/gscho/gemfast/internal/middleware"
	"github.com/rs/zerolog/log"
)

func Run() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	configureAuth(r)
	addRoutes(r)
	port := ":" + config.Env.Port
	log.Info().Str("port", port).Msg("gemfast server ready")
	return r.Run(port)
}

func configureAuth(r *gin.Engine) {
	authMode := config.Env.AuthMode
	switch strings.ToLower(authMode) {
	case "local":
		log.Info().Str("auth", authMode).Msg("configuring auth strategy")
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
	authMiddleware, err := initJwtMiddleware()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize auth middleware")
	}
	r.POST("/login", authMiddleware.LoginHandler)
	localAuth := r.Group("/")
	localAuth.GET("/refresh_token", authMiddleware.RefreshHandler)
	// authorized.Use(authMiddleware.MiddlewareFunc())
	// {
	// 	authorized.POST("/api/v1/gems", uploadGem)
	// 	authorized.POST("/upload", geminaboxUploadGem)
	// }
}

func configureNoneAuth(r *gin.Engine) {
	r.POST("/api/v1/gems", uploadGem)
	r.POST("/upload", geminaboxUploadGem)
}

func addRoutes(r *gin.Engine) {
	r.HEAD("/", head)
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
		tokenAuth.POST("create_token", createToken)
	}
}
