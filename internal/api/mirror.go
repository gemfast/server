package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/models"
	"github.com/gemfast/server/internal/spec"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func mirroredGemspecRzHandler(c *gin.Context) {
	fileName := c.Param("gemspec.rz")
	gemAllowed := filter.IsAllowed(fileName)
	if !gemAllowed {
		c.String(http.StatusForbidden, fmt.Sprintf("Refusing to download gemspec %s due to filter", fileName))
		return
	}
	fp := filepath.Join(config.Env.Dir, "quick/Marshal.4.8", fileName)
	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(fp)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to create gem file")
			return
		}
		defer out.Close()
		path, err := url.JoinPath(config.Env.MirrorUpstream, "quick/Marshal.4.8", fileName)
		if err != nil {
			log.Error().Str("file", fileName).Msg("failed to fetch quick marshal")
			panic(err)
		}
		resp, err := http.Get(path)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to connect to upstream")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Info().Str("upstream", path).Msg("upstream returned a non 200 status code")
			c.String(resp.StatusCode, "Failure returned from upstream")
			return
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to write gem file")
			return
		}
	}
	c.FileAttachment(fp, fileName)
}

func mirroredGemHandler(c *gin.Context) {
	fileName := c.Param("gem")
	gemAllowed := filter.IsAllowed(fileName)
	if !gemAllowed {
		c.String(http.StatusForbidden, fmt.Sprintf("Refusing to download gemspec %s due to filter", fileName))
		return
	}
	fp := filepath.Join(config.Env.GemDir, fileName)
	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(fp)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to create gem file")
		}
		defer out.Close()
		path, err := url.JoinPath(config.Env.MirrorUpstream, "gems", fileName)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to fetch gem file")
			return
		}
		resp, err := http.Get(path)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to connect to upstream")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Info().Str("upstream", path).Msg("upstream returned a non 200 status code")
			c.String(resp.StatusCode, "Failure returned from upstream")
			return
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to write gem file")
			return
		}
		s := spec.FromFile(fp)
		err = models.SetGem(s.Name, s.Version, s.OriginalPlatform)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to save gem in db")
			return
		}
		go indexer.Get().UpdateIndex()
	}
	c.FileAttachment(fp, fileName)
}

func mirroredIndexHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}

func mirroredInfoHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}

func mirroredVersionsHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}

func mirroredDependenciesHandler(c *gin.Context) {
	gemQuery := c.Query("gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
	path += "?gems="
	path += gemQuery
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}

func mirroredDependenciesJSONHandler(c *gin.Context) {
	gemQuery := c.Query("gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
	path += "?gems="
	path += gemQuery
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}
