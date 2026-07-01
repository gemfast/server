package api

import (
	"context"
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
	"github.com/gemfast/server/internal/telemetry"
	"github.com/gemfast/server/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// upstreamHTTPClient propagates trace context and emits a client span per call
// to the upstream mirror (rubygems.org by default).
var upstreamHTTPClient = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

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
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(
		attribute.String("gem.filename", fileName),
		attribute.String("gem.source", h.cfg.PrivateGemsNamespace),
	)
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
	span := trace.SpanFromContext(c.Request.Context())
	gemQuery := c.Query("gems")
	span.SetAttributes(
		attribute.String("gem.source", h.cfg.PrivateGemsNamespace),
		attribute.Int("gem.query.count", strings.Count(gemQuery, ",")+1),
	)
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
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(
		attribute.String("upload.path", "rubygems-push"),
		attribute.String("gem.source", h.cfg.PrivateGemsNamespace),
	)
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = io.ReadAll(c.Request.Body)
	}
	span.SetAttributes(attribute.Int("upload.size_bytes", len(bodyBytes)))
	tmpfile, err := os.CreateTemp(h.cfg.Dir+"/tmp", "*.gem")
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("exception.slug", "err-upload-create-tmpfile"))
		log.Error().Err(err).Msg("failed to create tmp file")
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to index gem: %v", err))
		return
	}
	defer os.Remove(tmpfile.Name())

	err = os.WriteFile(tmpfile.Name(), bodyBytes, 0644)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("exception.slug", "err-upload-write-tmpfile"))
		log.Error().Err(err).Str("detail", tmpfile.Name()).Msg("failed to save uploaded file")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to index gem: %v", err))
		return
	}
	if err = h.saveAndReindexLocalGem(c.Request.Context(), h.cfg.PrivateGemsNamespace, tmpfile); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("exception.slug", "err-upload-save-reindex"))
		log.Error().Err(err).Msg("failed to reindex gem")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to index gem: %v", err))
		return
	}
	c.String(http.StatusOK, "uploaded successfully")
}

func (h *RubyGemsHandler) localYankHandler(c *gin.Context) {
	span := trace.SpanFromContext(c.Request.Context())
	g := c.Query("gem")
	v := c.Query("version")
	p := c.Query("platform")
	span.SetAttributes(
		attribute.String("gem.name", g),
		attribute.String("gem.version", v),
		attribute.String("gem.platform", p),
		attribute.String("gem.source", h.cfg.PrivateGemsNamespace),
	)
	if g == "" || v == "" {
		span.SetAttributes(attribute.String("exception.slug", "err-yank-missing-params"))
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
	span := trace.SpanFromContext(c.Request.Context())
	span.SetAttributes(
		attribute.String("upload.path", "geminabox"),
		attribute.String("gem.source", h.cfg.PrivateGemsNamespace),
	)
	file, err := c.FormFile("file")
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("exception.slug", "err-upload-form-file-missing"))
		log.Error().Err(err).Msg("failed to read form file")
		c.String(http.StatusBadRequest, "failed to read form file parameter")
		return
	}
	span.SetAttributes(
		attribute.String("upload.filename", file.Filename),
		attribute.Int64("upload.size_bytes", file.Size),
	)
	tmpfile, err := os.CreateTemp(h.cfg.Dir+"/tmp", "*.gem")
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("exception.slug", "err-upload-create-tmpfile"))
		log.Error().Err(err).Msg("failed to create tmp file")
		c.String(http.StatusInternalServerError, "failed to index gem")
		return
	}
	defer os.Remove(tmpfile.Name())

	if err = c.SaveUploadedFile(file, tmpfile.Name()); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("exception.slug", "err-upload-save-uploaded-file"))
		log.Error().Err(err).Str("detail", tmpfile.Name()).Msg("failed to save uploaded file")
		c.String(http.StatusInternalServerError, "failed to index gem")
		return
	}
	if err = h.saveAndReindexLocalGem(c.Request.Context(), h.cfg.PrivateGemsNamespace, tmpfile); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("exception.slug", "err-upload-save-reindex"))
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

func (h *RubyGemsHandler) saveAndReindexLocalGem(ctx context.Context, source string, tmpfile *os.File) error {
	ctx, span := telemetry.Tracer().Start(ctx, "rubygems.saveAndReindexLocalGem")
	defer span.End()
	span.SetAttributes(
		attribute.String("gem.source", source),
		attribute.String("gem.action", "upload"),
	)
	s, err := spec.FromFile(h.cfg.Dir+"/tmp", tmpfile.Name())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "spec.FromFile failed")
		span.SetAttributes(attribute.String("exception.slug", "err-save-reindex-spec-parse"))
		log.Error().Err(err).Msg("failed to read spec from tmpfile")
		return err
	}
	span.SetAttributes(
		attribute.String("gem.name", s.Name),
		attribute.String("gem.version", s.Version),
		attribute.String("gem.platform", s.OriginalPlatform),
	)
	span.AddEvent("spec.parsed", trace.WithAttributes(
		attribute.String("gem.name", s.Name),
		attribute.String("gem.version", s.Version),
	))
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "rename tmpfile failed")
		span.SetAttributes(attribute.String("exception.slug", "err-save-reindex-rename"))
		log.Error().Err(err).Str("detail", fp).Msg("failed to rename tmpfile")
		return err
	}
	span.AddEvent("gem.file.persisted", trace.WithAttributes(
		attribute.String("path", fp),
	))
	span.AddEvent("indexer.add.start")
	err = h.indexer.AddGemToIndex(source, fp)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "AddGemToIndex failed")
		span.SetAttributes(attribute.String("exception.slug", "err-save-reindex-add-index"))
		log.Error().Err(err).Str("detail", s.Name).Msg("failed to add gem to index")
		return err
	}
	span.AddEvent("indexer.add.complete")
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
	span := trace.SpanFromContext(c.Request.Context())
	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
		span.SetAttributes(
			attribute.Bool("mirror.cache_hit", false),
			attribute.String("mirror.source", "upstream"),
		)
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
		req, err := http.NewRequestWithContext(c.Request.Context(), "GET", path, nil)
		if err != nil {
			log.Error().Err(err).Str("detail", path).Msg("failed to build upstream request")
			c.String(http.StatusInternalServerError, "Failed to build upstream request")
			return
		}
		resp, err := upstreamHTTPClient.Do(req)
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
		span.SetAttributes(
			attribute.Bool("mirror.cache_hit", true),
			attribute.String("mirror.source", "local"),
		)
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
	span := trace.SpanFromContext(c.Request.Context())
	info, err := os.Stat(fp)
	if (err != nil && errors.Is(err, os.ErrNotExist)) || info.Size() == 0 {
		span.SetAttributes(
			attribute.Bool("mirror.cache_hit", false),
			attribute.String("mirror.source", "upstream"),
		)
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
		req, err := http.NewRequestWithContext(c.Request.Context(), "GET", path, nil)
		if err != nil {
			log.Error().Err(err).Str("detail", path).Msg("failed to build upstream request")
			c.String(http.StatusInternalServerError, "Failed to build upstream request")
			return
		}
		resp, err := upstreamHTTPClient.Do(req)
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
		span.SetAttributes(
			attribute.Bool("mirror.cache_hit", true),
			attribute.String("mirror.source", "local"),
		)
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
