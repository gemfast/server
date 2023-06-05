package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/models"
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
	if config.Cfg.CVE.Enabled {
		gv := strings.Split(fileName, ".gemspec.rz")
		gem := models.GemFromGemParameter(gv[0])
		cves := cve.GetCVEs(gem.Name, gem.Number)
		if len(cves) != 0 {
			c.String(http.StatusForbidden, fmt.Sprintf("Refusing to download gem %s due to CVE: %s", fileName, cves[0].URL))
			return
		}
	}
	fp := filepath.Join(config.Cfg.Dir, "quick/Marshal.4.8", fileName)
	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(fp)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to create gem file")
			return
		}
		defer out.Close()
		path, err := url.JoinPath(config.Cfg.Mirrors[0].Upstream, "quick/Marshal.4.8", fileName)
		if err != nil {
			log.Error().Str("detail", fileName).Msg("failed to fetch quick marshal")
			panic(err)
		}
		resp, err := http.Get(path)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to connect to upstream")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Info().Str("detail", path).Msg("upstream returned a non 200 status code")
			c.String(resp.StatusCode, "Failure returned from upstream")
			out.Close()
			os.RemoveAll(fp)
			return
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to write gem file")
			return
		}
	} else {
		log.Trace().Msg("serving existing gemspec.rz")
	}
	c.FileAttachment(fp, fileName)
}

func mirroredGemHandler(c *gin.Context) {
	fileName := c.Param("gem")
	gemAllowed := filter.IsAllowed(fileName)
	if !gemAllowed {
		c.String(http.StatusForbidden, fmt.Sprintf("Refusing to download gems %s due to filter", fileName))
		return
	}
	if config.Cfg.CVE.Enabled {
		gv := strings.Split(fileName, ".gem")
		gem := models.GemFromGemParameter(gv[0])
		cves := cve.GetCVEs(gem.Name, gem.Number)
		if len(cves) != 0 {
			c.String(http.StatusForbidden, fmt.Sprintf("Refusing to download gem %s due to CVE", fileName))
			return
		}
	}
	fp := filepath.Join(config.Cfg.GemDir, fileName)
	info, err := os.Stat(fp)
	if (err != nil && errors.Is(err, os.ErrNotExist)) || info.Size() == 0 {
		out, err := os.Create(fp)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to create gem file")
		}
		defer out.Close()
		path, err := url.JoinPath(config.Cfg.Mirrors[0].Upstream, "gems", fileName)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to fetch gem file from upstream")
			return
		}
		resp, err := http.Get(path)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to connect to upstream")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Info().Str("detail", path).Msg("upstream returned a non 200 status code")
			c.String(resp.StatusCode, "Failure returned from upstream")
			return
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to write gem file")
			return
		}
		out.Close()
		err = indexer.Get().AddGemToIndex(fp)
		if err != nil {
			defer os.Remove(fp)
			c.String(http.StatusInternalServerError, "Failed to index gem")
			return
		}
	} else {
		log.Trace().Msg("serving existing gem")
	}
	c.FileAttachment(fp, fileName)
}

func mirroredIndexHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Cfg.Mirrors[0].Upstream, c.FullPath())
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}

func mirroredInfoHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Cfg.Mirrors[0].Upstream, c.FullPath())
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}

func mirroredVersionsHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Cfg.Mirrors[0].Upstream, c.FullPath())
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
	path, err := url.JoinPath(config.Cfg.Mirrors[0].Upstream, c.FullPath())
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
	path, err := url.JoinPath(config.Cfg.Mirrors[0].Upstream, c.FullPath())
	path += "?gems="
	path += gemQuery
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}
