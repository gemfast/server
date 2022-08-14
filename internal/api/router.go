package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func Run() error {
	r := gin.Default()
	addRoutes(r)
	return r.Run()
}

func addRoutes(r *gin.Engine) {
	r.HEAD("/", head)
	fmt.Println(fmt.Sprintf("%s/specs.4.8.gz", viper.Get("dir")))
	r.StaticFile("/specs.4.8.gz", fmt.Sprintf("%s/specs.4.8.gz", viper.Get("dir")))
	r.StaticFile("/latest_specs.4.8.gz", fmt.Sprintf("%s/latest_specs.4.8.gz", viper.Get("dir")))
	r.StaticFile("/prerelease_specs.4.8.gz", fmt.Sprintf("%s/prerelease_specs.4.8.gz", viper.Get("dir")))
	r.GET("/quick/Marshal.4.8/*gemspec.rz", getGemspecRz)
	r.GET("/gems/*gem", getGem)
	r.POST("/api/v1/gems", uploadGem)
	r.POST("/upload", geminaboxUploadGem)
	// r.GET("/api/v1/dependencies", getDependencies)
	r.GET("/api/v1/dependencies.json", getDependenciesJSON)
}
