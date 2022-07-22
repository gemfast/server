package main

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/gscho/gemfast/internal/indexer"
	"github.com/gscho/gemfast/internal/spec"
)

type Gem struct {
	Version string
}

func main() {
	i := indexer.New("/var/gemfast")
	i.GenerateIndex()
	r := gin.Default()
	r.HEAD("/", func(c *gin.Context) {})
	r.StaticFile("/specs.4.8.gz", "/var/gemfast/specs.4.8.gz")
	r.StaticFile("/latest_specs.4.8.gz", "/var/gemfast/latest_specs.4.8.gz")
	r.StaticFile("/prerelease_specs.4.8.gz", "/var/gemfast/prerelease_specs.4.8.gz")
	// r.StaticFile("/quick/Marshal.4.8/mixlib-install-3.0.0.gemspec.rz", "/var/gemfast/quick/Marshal.4.8/mixlib-install-3.0.0.gemspec.rz")
	r.POST("/api/v1/gems", func(c *gin.Context) {
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
		}
		file, err := ioutil.TempFile("/tmp", "*.gem")
		if err != nil {
		    panic(err)
		}
		defer os.Remove(file.Name())

		err = os.WriteFile(file.Name(), bodyBytes, 0644)
		if err != nil {
			panic(err)
		}
		s := spec.FromFile(file.Name())
		err = os.Rename(file.Name(), fmt.Sprintf("/var/gemfast/%s-%s.gem", s.Name, s.Version))
		if err != nil {
			panic(err)
		}
		i.UpdateIndex()
		c.String(http.StatusOK, "derpydo")
	})
	r.Run()
}
