package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/marshal"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func localGemspecRzHandler(c *gin.Context) {
	fileName := c.Param("gemspec.rz")
	fp := filepath.Join(config.Env.Dir, "quick/Marshal.4.8", fileName)
	c.FileAttachment(fp, fileName)
}

func localGemHandler(c *gin.Context) {
	fileName := c.Param("gem")
	fp := filepath.Join(config.Env.GemDir, fileName)
	c.FileAttachment(fp, fileName)
}

func localIndexHandler(c *gin.Context) {
	c.File(filepath.Join(config.Env.Dir, c.FullPath()))
}

func localDependenciesHandler(c *gin.Context) {
	gemQuery := c.Query("gems")
	log.Trace().Str("gems", gemQuery).Msg("received gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	deps, err := fetchGemDependencies(c, gemQuery)
	if err != nil && config.Env.MirrorEnabled != "false" {
		c.String(http.StatusNotFound, fmt.Sprintf("failed to fetch dependencies for gem: %s", gemQuery))
		return
	} else if err != nil && config.Env.MirrorEnabled != "false" {
		path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
		path += "?gems="
		path += gemQuery
		if err != nil {
			panic(err)
		}
		c.Redirect(http.StatusFound, path)
	}
	bundlerDeps, err := marshal.DumpBundlerDeps(deps)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal gem dependencies")
		c.String(http.StatusInternalServerError, "failed to marshal gem dependencies")
		return
	}
	c.Header("Content-Type", "application/octet-stream; charset=utf-8")
	c.Writer.Write(bundlerDeps)
}

func localDependenciesJSONHandler(c *gin.Context) {
	gemQuery := c.Query("gems")
	log.Trace().Str("gems", gemQuery).Msg("received gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	deps, err := fetchGemDependencies(c, gemQuery)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, deps)
}

func localUploadGemHandler(c *gin.Context) {
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	tmpfile, err := ioutil.TempFile("/tmp", "*.gem")
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmp file")
		c.String(http.StatusInternalServerError, "Failed to index gem")
		return
	}
	defer os.Remove(tmpfile.Name())

	err = os.WriteFile(tmpfile.Name(), bodyBytes, 0644)
	if err != nil {
		log.Error().Err(err).Str("tmpfile", tmpfile.Name()).Msg("failed to save uploaded file")
		c.String(http.StatusInternalServerError, "failed to index gem")
		return
	}
	if err = saveAndReindex(tmpfile); err != nil {
		log.Error().Err(err).Msg("failed to reindex gem")
		c.String(http.StatusInternalServerError, "failed to index gem")
		return
	}
	c.String(http.StatusOK, "uploaded successfully")
}
