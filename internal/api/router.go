package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	cfg "github.com/gscho/gemfast/internal/config"
)

var r *gin.Engine

func init() {
	r = gin.Default()
	addRoutes(r)
}

func Run() error {
	return r.Run()
}

func addRoutes(r *gin.Engine) {
	r.HEAD("/", head)
	r.StaticFile("/specs.4.8.gz", fmt.Sprintf("%s/specs.4.8.gz", cfg.Get("dir")))
	r.StaticFile("/latest_specs.4.8.gz", fmt.Sprintf("%s/latest_specs.4.8.gz", cfg.Get("dir")))
	r.StaticFile("/prerelease_specs.4.8.gz", fmt.Sprintf("%s/prerelease_specs.4.8.gz", cfg.Get("dir")))
	r.GET("/quick/Marshal.4.8/*gemspec.rz", getGemspecRz)
	r.GET("/gems/*gem", getGem)
	r.POST("/api/v1/gems", uploadGem)
	r.POST("/upload", geminaboxUploadGem)
}
