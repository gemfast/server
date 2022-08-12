package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gscho/gemfast/internal/indexer"
	"github.com/gscho/gemfast/internal/spec"
	"github.com/spf13/viper"
)

func head(c *gin.Context) {}

func getGemspecRz(c *gin.Context) {
	fileName := c.Param("gemspec.rz")
	filePath := fmt.Sprintf("%s/quick/Marshal.4.8/%s", viper.Get("dir"), fileName)
	c.FileAttachment(filePath, fileName)
}

func getGem(c *gin.Context) {
	fileName := c.Param("gem")
	filePath := fmt.Sprintf("%s/%s", viper.Get("dir"), fileName)
	c.FileAttachment(filePath, fileName)
}

func saveAndReindex(tmpfile *os.File) (error) {
	s := spec.FromFile(tmpfile.Name())
	filePath := fmt.Sprintf("%s/%s-%s.gem", viper.Get("dir"), s.Name, s.Version)
	err := os.Rename(tmpfile.Name(), filePath)
	go indexer.Get().UpdateIndex()
	return err
}

func uploadGem(c *gin.Context) {
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	tmpfile, err := ioutil.TempFile("/tmp", "*.gem")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	err = os.WriteFile(tmpfile.Name(), bodyBytes, 0644)
	if err != nil {
		panic(err)
	}
	if err = saveAndReindex(tmpfile); err != nil {
		panic(err)
	}
	c.String(http.StatusOK, "Uploaded successfully")
}

func geminaboxUploadGem(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		panic(err)
	}
	tmpfile, err := ioutil.TempFile("/tmp", "*.gem")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpfile.Name())

	err = c.SaveUploadedFile(file, tmpfile.Name())
	if err != nil {
		panic(err)
	}
	if err = saveAndReindex(tmpfile); err != nil {
		panic(err)
	}
	c.String(http.StatusOK, "Uploaded successfully")
}
