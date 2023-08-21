package api

import (
	"errors"
	"fmt"
	"io"
	"path"

	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/marshal"
	"github.com/gemfast/server/internal/spec"
	"github.com/gemfast/server/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type RubyGemsHandler struct {
	cfg        *config.Config
	db         *db.DB
	indexer    *indexer.Indexer
	Filter     *filter.RegexFilter
	advisoryDB *cve.GemAdvisoryDB
}

func NewRubyGemsHandler(cfg *config.Config, database *db.DB, i *indexer.Indexer, f *filter.RegexFilter, advisoryDB *cve.GemAdvisoryDB) *RubyGemsHandler {
	return &RubyGemsHandler{
		cfg:        cfg,
		db:         database,
		indexer:    i,
		Filter:     f,
		advisoryDB: advisoryDB,
	}
}

type BundlerDeps struct {
	Name         string
	Number       string
	Platform     string
	Dependencies [][]string
}

func newBundlerDeps(g *db.Gem) (*BundlerDeps, error) {
	b := &BundlerDeps{
		Name:     g.Name,
		Number:   g.Number,
		Platform: g.Platform,
	}
	var deps [][]string
	for _, d := range g.Dependencies {
		if d.Type == ":runtime" {
			deps = append(deps, []string{d.Name, d.VersionConstraints})
		}
	}
	b.Dependencies = deps
	return b, nil
}

func (h *RubyGemsHandler) localGemspecRzHandler(c *gin.Context) {
	fileName := c.Param("gemspec.rz")
	fp := filepath.Join(h.cfg.Dir, "quick/Marshal.4.8", fileName)
	c.FileAttachment(fp, fileName)
}

func (h *RubyGemsHandler) localGemHandler(c *gin.Context) {
	fileName := c.Param("gem")
	fc := strings.Split(fileName, "")[0] // first character
	fp := filepath.Join(h.cfg.GemDir, h.cfg.PrivateGemsNamespace, fc, fileName)
	c.FileAttachment(fp, fileName)
}

func (h *RubyGemsHandler) localIndexHandler(c *gin.Context) {
	s := strings.Split(c.FullPath(), "/")
	l := len(s)
	c.File(filepath.Join(h.cfg.Dir, s[l-1]))
}

func (h *RubyGemsHandler) localDependenciesHandler(c *gin.Context) {
	gemQuery := c.Query("gems")
	log.Trace().Str("detail", gemQuery).Msg("received gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	gemVersions, err := h.fetchGemVersions(h.cfg.PrivateGemsNamespace, gemQuery)
	if err != nil && !h.cfg.Mirrors[0].Enabled {
		c.String(http.StatusNotFound, fmt.Sprintf("failed to fetch dependencies for gem: %s", gemQuery))
		return
	} else if err != nil && h.cfg.Mirrors[0].Enabled {
		path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, c.FullPath())
		if err != nil {
			log.Error().Err(err).Msg("failed to join upstream path")
			c.String(http.StatusInternalServerError, fmt.Sprintf("failed to join upstream path: %v", err))
			return
		}
		path += "?gems="
		path += gemQuery

		c.Redirect(http.StatusFound, path)
	}
	bundlerDeps, err := marshal.DumpBundlerDeps(gemVersions)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal gem dependencies")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal gem dependencies: %v", err))
		return
	}
	c.Header("Content-Type", "application/octet-stream; charset=utf-8")
	c.Writer.Write(bundlerDeps)
}

func (h *RubyGemsHandler) localDependenciesJSONHandler(c *gin.Context) {
	gemQuery := c.Query("gems")
	log.Trace().Str("detail", gemQuery).Msg("received gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	gemVersions, err := h.fetchGemVersions(h.cfg.PrivateGemsNamespace, gemQuery)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch gem versions")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to fetch gem versions: %v", err))
		return
	}
	var deps []*BundlerDeps
	for _, gv := range gemVersions {
		bundlerDep, err := newBundlerDeps(gv)
		if err != nil {
			log.Error().Err(err).Msg("failed to create new bundler deps")
			c.String(http.StatusInternalServerError, fmt.Sprintf("failed to create new bundler deps: %v", err))
			return
		}
		deps = append(deps, bundlerDep)
	}
	c.JSON(http.StatusOK, deps)
}

func (h *RubyGemsHandler) localUploadGemHandler(c *gin.Context) {
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = io.ReadAll(c.Request.Body)
	}
	tmpfile, err := os.CreateTemp("/tmp", "*.gem")
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmp file")
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to index gem: %v", err))
		return
	}
	defer os.Remove(tmpfile.Name())

	err = os.WriteFile(tmpfile.Name(), bodyBytes, 0644)
	if err != nil {
		log.Error().Err(err).Str("detail", tmpfile.Name()).Msg("failed to save uploaded file")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to index gem: %v", err))
		return
	}
	if err = h.saveAndReindexLocalGem(h.cfg.PrivateGemsNamespace, tmpfile); err != nil {
		log.Error().Err(err).Msg("failed to reindex gem")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to index gem: %v", err))
		return
	}
	c.String(http.StatusOK, "uploaded successfully")
}

func (h *RubyGemsHandler) localYankHandler(c *gin.Context) {
	g := c.Query("gem")
	v := c.Query("version")
	p := c.Query("platform")
	if g == "" || v == "" {
		c.String(http.StatusBadRequest, "must provide both gem and version query parameters")
		return
	}
	err := h.indexer.RemoveGemFromIndex(g, v, p)
	if err != nil {
		log.Error().Err(err).Msg("failed to yank gem from index")
		c.String(http.StatusInternalServerError, fmt.Sprintf("server failed to yank gem from index: %v", err))
		return
	}
	fileName := g + "-" + v + ".gem"
	fp := filepath.Join(h.cfg.GemDir, h.cfg.PrivateGemsNamespace, fileName)
	err = utils.RemoveFileIfExists(fp)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete gem file system")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete gem from file system: %v", err))
		return
	}
	fileName = fileName + "spec.rz"
	fp = filepath.Join(h.cfg.Dir, "quick/Marshal.4.8", fileName)
	err = utils.RemoveFileIfExists(fp)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete gemspec.rz from file system")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete gem file system: %v", err))
		return
	}
	num, err := h.db.DeleteGemVersion(h.cfg.PrivateGemsNamespace, &db.Gem{Name: g, Number: v, Platform: p})
	if err != nil {
		log.Error().Err(err).Msg("failed to yank gem")
		c.String(http.StatusInternalServerError, fmt.Sprintf("server failed to yank gem: %v", err))
		return
	}
	if num == 0 {
		c.String(http.StatusNotFound, "no gem matching %s-%s-%s found", g, v, p)
		return
	}
	c.String(http.StatusOK, "successfully yanked")
}

func (h *RubyGemsHandler) localVersionsHandler(c *gin.Context) {
	versions, err := h.db.GetAllGemversions(h.cfg.PrivateGemsNamespace)
	if err != nil {
		log.Error().Err(err).Msg("failed to get all gem versions")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get all gem versions: %v", err))
		return
	}
	c.String(http.StatusOK, strings.Join(versions, "\n"))
}

func (h *RubyGemsHandler) localNamesHandler(c *gin.Context) {
	names := h.db.GetAllGemNames(h.cfg.PrivateGemsNamespace)
	c.String(http.StatusOK, (strings.Join(names, "\n") + "\n"))
}

func (h *RubyGemsHandler) localInfoHandler(c *gin.Context) {
	gem := c.Param("gem")
	if gem == "" {
		c.String(http.StatusBadRequest, "must provide gem name")
		return
	}
	info, err := h.db.GetGemInfo(h.cfg.PrivateGemsNamespace, gem)
	if err != nil {
		log.Error().Err(err).Msg("failed to get gem info")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get gem info: %v", err))
		return
	}
	c.String(http.StatusOK, info+"\n")
}

func (h *RubyGemsHandler) geminaboxUploadGem(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		log.Error().Err(err).Msg("failed to read form file")
		c.String(http.StatusBadRequest, "failed to read form file parameter")
		return
	}
	tmpfile, err := os.CreateTemp("/tmp", "*.gem")
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmp file")
		c.String(http.StatusInternalServerError, "failed to index gem")
		return
	}
	defer os.Remove(tmpfile.Name())

	if err = c.SaveUploadedFile(file, tmpfile.Name()); err != nil {
		log.Error().Err(err).Str("detail", tmpfile.Name()).Msg("failed to save uploaded file")
		c.String(http.StatusInternalServerError, "failed to index gem")
		return
	}
	if err = h.saveAndReindexLocalGem(h.cfg.PrivateGemsNamespace, tmpfile); err != nil {
		log.Error().Err(err).Msg("failed to reindex gem")
		c.String(http.StatusInternalServerError, "failed to index gem")
		return
	}
	c.String(http.StatusOK, "uploaded successfully")
}

func (h *RubyGemsHandler) fetchGemVersions(source, gemQuery string) ([]*db.Gem, error) {
	gems := strings.Split(gemQuery, ",")
	var gemVersions []*db.Gem
	for _, gem := range gems {
		gv, err := h.db.GetGemVersions(source, gem)
		if err != nil {
			log.Trace().Err(err).Str("detail", gem).Msg("failed to fetch dependencies for gem")
			return nil, err
		}
		for _, g := range gv {
			gemVersions = append(gemVersions, &db.Gem{
				Name:         g.Name,
				Number:       g.Number,
				Dependencies: g.Dependencies,
			})
		}
	}
	return gemVersions, nil
}

func (h *RubyGemsHandler) saveAndReindexLocalGem(source string, tmpfile *os.File) error {
	s, err := spec.FromFile(tmpfile.Name())
	if err != nil {
		log.Error().Err(err).Msg("failed to read spec from tmpfile")
		return err
	}
	fc := strings.Split(s.Name, "")[0] // first character
	var fp string
	if s.OriginalPlatform == "ruby" {
		fp = fmt.Sprintf("%s/%s/%s/%s-%s.gem", h.cfg.GemDir, h.cfg.PrivateGemsNamespace, fc, s.Name, s.Version)
	} else {
		fp = fmt.Sprintf("%s/%s/%s/%s-%s-%s.gem", h.cfg.GemDir, h.cfg.PrivateGemsNamespace, fc, s.Name, s.Version, s.OriginalPlatform)
	}
	utils.MkDirs(path.Dir(fp))
	err = os.Rename(tmpfile.Name(), fp)
	if err != nil {
		log.Error().Err(err).Str("detail", fp).Msg("failed to rename tmpfile")
		return err
	}
	err = h.indexer.AddGemToIndex(source, fp)
	if err != nil {
		log.Error().Err(err).Str("detail", s.Name).Msg("failed to add gem to index")
		return err
	}
	return nil
}

func (h *RubyGemsHandler) mirroredGemspecRzHandler(c *gin.Context) {
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

func (h *RubyGemsHandler) mirroredGemHandler(c *gin.Context) {
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

func (h *RubyGemsHandler) mirroredIndexHandler(c *gin.Context) {
	path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, c.FullPath())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	c.Redirect(http.StatusFound, path)
}

func (h *RubyGemsHandler) mirroredInfoHandler(c *gin.Context) {
	gem := c.Param("gem")
	path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, "info", gem)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	c.Redirect(http.StatusFound, path)
}

func (h *RubyGemsHandler) mirroredVersionsHandler(c *gin.Context) {
	path, err := url.JoinPath(h.cfg.Mirrors[0].Upstream, c.FullPath())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	c.Redirect(http.StatusFound, path)
}

func (h *RubyGemsHandler) mirroredDependenciesHandler(c *gin.Context) {
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

func (h *RubyGemsHandler) mirroredDependenciesJSONHandler(c *gin.Context) {
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
