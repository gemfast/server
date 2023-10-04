package ui

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/spec"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

//go:embed templates/*
var templates embed.FS

//go:embed assets/*
var assets embed.FS

type UI struct {
	cfg       *config.Config
	db        *db.DB
	Templates *template.Template
	Assets    fs.FS
}

func NewUI(cfg *config.Config, db *db.DB) *UI {
	static, err := fs.Sub(assets, "assets")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load assets")
	}
	tmpl := template.Must(template.New("").ParseFS(templates, "templates/**/*.tmpl", "templates/**/**/*.tmpl"))
	return &UI{cfg: cfg, db: db, Templates: tmpl, Assets: static}
}

func (ui *UI) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index", nil)
}

func (ui *UI) Gems(c *gin.Context) {
	entries, err := os.ReadDir(ui.cfg.GemDir)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	dirs := []string{}
	for _, entry := range entries {
		dirs = append(dirs, entry.Name())
	}
	c.HTML(http.StatusOK, "gems", gin.H{
		"sources": dirs,
	})
}

func (ui *UI) GemsOptions(c *gin.Context) {
	c.String(http.StatusOK, "GET")
}

func (ui *UI) GemsByPrefix(c *gin.Context) {
	source := c.Param("source")
	fp := filepath.Join(ui.cfg.GemDir, source)
	entries, err := os.ReadDir(ui.cfg.GemDir + "/" + source)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	dirs := make(map[string]int)
	for _, entry := range entries {
		child, err := os.ReadDir(fp + "/" + entry.Name())
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		dirs[entry.Name()] = len(child)
	}
	c.HTML(http.StatusOK, "gems/prefix", gin.H{
		"dirs":   dirs,
		"source": source,
	})
}

func (ui *UI) GemsData(c *gin.Context) {
	source := c.Param("source")
	prefix := c.Param("prefix")
	gems := ui.db.PrefixScanGems(source, prefix)
	c.HTML(http.StatusOK, "gems/data", gin.H{
		"gems":   gems,
		"source": source,
		"prefix": prefix,
	})
}

func (ui *UI) GemsInspect(c *gin.Context) {
	source := c.Param("source")
	prefix := c.Param("prefix")
	gem := c.Param("gem")
	version := c.Query("version")
	gemVersions, err := ui.db.GetGemVersions(source, gem)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if len(gemVersions) == 0 {
		c.String(http.StatusNotFound, "gem not found")
		return
	}
	sort.Slice(gemVersions, func(i, j int) bool {
		return gemVersions[i].Number > gemVersions[j].Number
	})
	var gv *db.Gem
	if version == "" {
		gv = gemVersions[0]
	} else {
		for _, g := range gemVersions {
			if g.Number == version {
				gv = g
			}
		}
	}
	gemFName := gv.Name + "-" + gv.Number + ".gem"
	gemfile := filepath.Join(ui.cfg.GemDir, source, prefix, gemFName)
	spec, err := spec.FromFile(ui.cfg.Dir+"/tmp", gemfile)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.HTML(http.StatusOK, "gems/inspect", gin.H{
		"gemVersions": gemVersions,
		"source":      source,
		"prefix":      prefix,
		"gem":         gem,
		"spec":        spec,
		"gv":          gv,
	})
}

type SearchResults struct {
	Source string
	Name   string
}

func (ui *UI) SearchGems(c *gin.Context) {
	gem := c.PostForm("gemName")
	private := ui.db.SearchGems(ui.cfg.PrivateGemsNamespace, gem)
	rubygems := ui.db.SearchGems(ui.cfg.Mirrors[0].Hostname, gem)
	var gems []*SearchResults
	for _, g := range private {
		gems = append(gems, &SearchResults{Source: ui.cfg.PrivateGemsNamespace, Name: g})
	}
	for _, g := range rubygems {
		gems = append(gems, &SearchResults{Source: ui.cfg.Mirrors[0].Hostname, Name: g})
	}
	c.HTML(http.StatusOK, "gems/search", gin.H{
		"gems": gems,
	})
}

func (ui *UI) UploadGem(c *gin.Context) {
	c.HTML(http.StatusOK, "upload", nil)
}

func (ui *UI) License(c *gin.Context) {
	l, err := ui.db.GetLicense()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.HTML(http.StatusOK, "license", gin.H{
		"license": l,
	})
}
