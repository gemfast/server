package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type RubyGemsMirrorHandler struct {
	cfg        *config.Config
	db         *db.DB
	indexer    *indexer.Indexer
	Filter     *filter.RegexFilter
	advisoryDB *cve.GemAdvisoryDB
}

func NewRubyGemsMirrorHandler(cfg *config.Config, database *db.DB, i *indexer.Indexer, f *filter.RegexFilter, advisoryDB *cve.GemAdvisoryDB) *RubyGemsMirrorHandler {
	return &RubyGemsMirrorHandler{
		cfg:        cfg,
		db:         database,
		indexer:    i,
		Filter:     f,
		advisoryDB: advisoryDB,
	}
}
func (h *RubyGemsMirrorHandler) GetGemspecRz(c *gin.Context) {
	fileName := c.Param("gemspec.rz")
	gemAllowed := h.Filter.IsAllowed(fileName)
	if !gemAllowed {
		c.String(http.StatusMethodNotAllowed, fmt.Sprintf("Refusing to download gemspec %s due to filter", fileName))
		return
	}
	if h.cfg.CVE.Enabled {
		gv := strings.Split(fileName, ".gemspec.rz")
		gem := db.GemFromGemParameter(gv[0])
		cves := h.advisoryDB.GetCVEs(gem.Name, gem.Number)
		if len(cves) != 0 {
			c.String(http.StatusMethodNotAllowed, fmt.Sprintf("Refusing to download gem %s due to CVE: %s", fileName, cves[0].URL))
			return
		}
	}
	fp := filepath.Join(h.cfg.Dir, "quick/Marshal.4.8", fileName)
	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(fp)
		if err != nil {
			log.Error().Err(err).Msg("failed to create gemspec.rz file")
			c.String(http.StatusInternalServerError, "Failed to create gemspec.rz file")
			return
		}
		defer out.Close()
		path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, "quick/Marshal.4.8", fileName)
		if err != nil {
			log.Error().Str("detail", fileName).Msg("failed to fetch quick marshal")
			c.String(http.StatusInternalServerError, "Failed to fetch quick marshal")
			return
		}
		resp, err := http.Get(path)
		if err != nil {
			log.Error().Err(err).Str("detail", path).Msg("failed to connect to upstream")
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
			log.Error().Err(err).Msg("failed to write gemspec.rz file")
			c.String(http.StatusInternalServerError, "Failed to write gemspec.rz file")
			return
		}
	} else {
		log.Trace().Msg("serving existing gemspec.rz")
	}
	c.FileAttachment(fp, fileName)
}

func (h *RubyGemsMirrorHandler) GetGem(c *gin.Context) {
	fileName := c.Param("gem")
	gemAllowed := h.Filter.IsAllowed(fileName)
	if !gemAllowed {
		c.String(http.StatusMethodNotAllowed, fmt.Sprintf("Refusing to download gems %s due to filter", fileName))
		return
	}
	if h.cfg.CVE.Enabled {
		gv := strings.Split(fileName, ".gem")
		gem := db.GemFromGemParameter(gv[0])
		cves := h.advisoryDB.GetCVEs(gem.Name, gem.Number)
		if len(cves) != 0 {
			c.String(http.StatusMethodNotAllowed, fmt.Sprintf("Refusing to download gem %s due to CVE", fileName))
			return
		}
	}
	fc := strings.Split(fileName, "")[0] // first character
	fp := filepath.Join(h.cfg.GemDir, h.cfg.Mirrors[0].Hostname, fc, fileName)
	info, err := os.Stat(fp)
	if (err != nil && errors.Is(err, os.ErrNotExist)) || info.Size() == 0 {
		utils.MkDirs(path.Dir(fp))
		out, err := os.Create(fp)
		if err != nil {
			log.Error().Err(err).Msg("failed to create gem file")
			c.String(http.StatusInternalServerError, "Failed to create gem file")
			return
		}
		defer out.Close()
		path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, "gems", fileName)
		if err != nil {
			log.Error().Err(err).Str("detail", fileName).Msg("failed to fetch gem file from upstream")
			c.String(http.StatusInternalServerError, "Failed to fetch gem file from upstream")
			return
		}
		resp, err := http.Get(path)
		if err != nil {
			log.Error().Err(err).Str("detail", path).Msg("failed to connect to upstream")
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
			log.Error().Err(err).Msg("failed to write gem file")
			c.String(http.StatusInternalServerError, "Failed to write gem file")
			return
		}
		out.Close()
		err = h.indexer.AddGemToIndex(h.cfg.Mirrors[0].Hostname, fp)
		if err != nil {
			defer os.Remove(fp)
			log.Error().Err(err).Msg("failed to index gem")
			c.String(http.StatusInternalServerError, "Failed to index gem")
			return
		}
	} else {
		log.Trace().Msg("serving existing gem")
	}
	c.FileAttachment(fp, fileName)
}

func (h *RubyGemsMirrorHandler) GetIndex(c *gin.Context) {
	path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, c.FullPath())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	c.Redirect(http.StatusFound, path)
}

func (h *RubyGemsMirrorHandler) GetGemInfo(c *gin.Context) {
	gem := c.Param("gem")
	path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, "info", gem)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	c.Redirect(http.StatusFound, path)
}

func (h *RubyGemsMirrorHandler) GetGemVersionsCompact(c *gin.Context) {
	path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, c.FullPath())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	c.Redirect(http.StatusFound, path)
}

func (h *RubyGemsMirrorHandler) GetGemDependencies(c *gin.Context) {
	gemQuery := c.Query("gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, c.FullPath())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	path += "?gems="
	path += gemQuery
	c.Redirect(http.StatusFound, path)
}

func (h *RubyGemsMirrorHandler) GetGemDependenciesJSON(c *gin.Context) {
	gemQuery := c.Query("gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, c.FullPath())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	path += "?gems="
	path += gemQuery
	c.Redirect(http.StatusFound, path)
}
