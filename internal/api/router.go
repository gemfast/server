package api

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gscho/gemfast/internal/models"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func Run() error {
	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	configureAuth(r)
	addRoutes(r)
	port := ":" + viper.GetString("port")
	log.Info().Str("port", port).Msg("gemfast server ready")
	return r.Run(port)
}

func configureAuth(r *gin.Engine) {
	authMode := viper.Get("auth").(string)
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
	authMiddleware, err := initAuthMiddleware()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize auth middleware")
	}
	r.POST("/login", authMiddleware.LoginHandler)
	authorized := r.Group("/")
	authorized.GET("/refresh_token", authMiddleware.RefreshHandler)
	authorized.Use(authMiddleware.MiddlewareFunc())
	{
		authorized.POST("/api/v1/gems", uploadGem)
		authorized.POST("/upload", geminaboxUploadGem)
	}
}

func configureNoneAuth(r *gin.Engine) {
	r.POST("/api/v1/gems", uploadGem)
	r.POST("/upload", geminaboxUploadGem)
}

func addRoutes(r *gin.Engine) {
	r.HEAD("/", head)
	r.StaticFile("/specs.4.8.gz", fmt.Sprintf("%s/specs.4.8.gz", viper.Get("dir")))
	r.StaticFile("/latest_specs.4.8.gz", fmt.Sprintf("%s/latest_specs.4.8.gz", viper.Get("dir")))
	r.StaticFile("/prerelease_specs.4.8.gz", fmt.Sprintf("%s/prerelease_specs.4.8.gz", viper.Get("dir")))
	r.GET("/quick/Marshal.4.8/*gemspec.rz", getGemspecRz)
	r.GET("/gems/*gem", getGem)
	r.GET("/api/v1/dependencies", getDependencies)
	r.GET("/api/v1/dependencies.json", getDependenciesJSON)
}
