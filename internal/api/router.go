package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func Run() error {
	r := gin.Default()
	addRoutes(r)
	return r.Run()
}

func addRoutes(r *gin.Engine) {
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
	r.HEAD("/", head)
	r.StaticFile("/specs.4.8.gz", fmt.Sprintf("%s/specs.4.8.gz", viper.Get("dir")))
	r.StaticFile("/latest_specs.4.8.gz", fmt.Sprintf("%s/latest_specs.4.8.gz", viper.Get("dir")))
	r.StaticFile("/prerelease_specs.4.8.gz", fmt.Sprintf("%s/prerelease_specs.4.8.gz", viper.Get("dir")))
	r.GET("/quick/Marshal.4.8/*gemspec.rz", getGemspecRz)
	r.GET("/gems/*gem", getGem)
	r.GET("/api/v1/dependencies", getDependencies)
	r.GET("/api/v1/dependencies.json", getDependenciesJSON)
}
