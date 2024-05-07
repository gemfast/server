package ui

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	jmw "github.com/appleboy/gin-jwt/v2"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/middleware"
	"github.com/gemfast/server/internal/spec"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
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

func getUser(c *gin.Context, ui *UI) (string, error) {
	if ui.cfg.Auth.Type == "github" {
		session := sessions.Default(c)
		sessionAuth := session.Get("authToken")
		jwtToken, err := jwt.Parse(sessionAuth.(string), func(t *jwt.Token) (interface{}, error) {
			if jwt.GetSigningMethod("HS256") != t.Method {
				return nil, errors.New("invalid signing method")
			}
			return []byte(ui.cfg.Auth.JWTSecretKey), nil
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to parse jwt token from request")
			return "", err
		}
		if !jwtToken.Valid {
			log.Error().Msg("invalid jwt token")
			return "", err
		}
		claims := jmw.ExtractClaimsFromToken(jwtToken)
		username, ok := claims[middleware.IdentityKey].(string)
		if !ok {
			log.Error().Str("username", username).Msg("failed to get user from jwt token")
			return "", err
		}
		return username, nil
	}
	return "anon", nil
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
	fmt.Println(c.GetHeader("reload"))
	if c.GetHeader("reload") == "true" {
		c.Header("HX-Redirect", "/ui")
		c.Abort()
		return
	}
	username, err := getUser(c, ui)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.HTML(http.StatusOK, "index", gin.H{
		"authType": ui.cfg.Auth.Type,
		"username": username,
	})
}

func (ui *UI) Gems(c *gin.Context) {
	if c.GetHeader("HX-Request") != "true" {
		c.Redirect(http.StatusFound, "/ui")
		return
	}
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
	if c.GetHeader("HX-Request") != "true" {
		c.Redirect(http.StatusFound, "/ui")
		return
	}
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
	if c.GetHeader("HX-Request") != "true" {
		c.Redirect(http.StatusFound, "/ui")
		return
	}
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
	if c.GetHeader("HX-Request") != "true" {
		c.Redirect(http.StatusFound, "/ui")
		return
	}
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
		"gemfastURL":  "http://localhost:2020",
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
	if c.GetHeader("HX-Request") != "true" {
		c.Redirect(http.StatusFound, "/ui")
		return
	}
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
	if c.GetHeader("HX-Request") != "true" {
		c.Redirect(http.StatusFound, "/ui")
		return
	}
	c.HTML(http.StatusOK, "upload", gin.H{})
}

func (ui *UI) AccessTokens(c *gin.Context) {
	if c.GetHeader("HX-Request") != "true" {
		c.Redirect(http.StatusFound, "/ui")
		return
	}
	session := sessions.Default(c)
	sessionAuth := session.Get("authToken")
	jwtToken, err := jwt.Parse(sessionAuth.(string), func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod("HS256") != t.Method {
			return nil, errors.New("invalid signing method")
		}
		return []byte(ui.cfg.Auth.JWTSecretKey), nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to parse jwt token from request")
	}
	if !jwtToken.Valid {
		log.Error().Msg("invalid jwt token")
	}
	claims := jmw.ExtractClaimsFromToken(jwtToken)
	username, ok := claims[middleware.IdentityKey].(string)
	if !ok {
		log.Error().Str("username", username).Msg("failed to get user from jwt token")
	}
	user, err := ui.db.GetUser(username)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user from db")
	}
	rubygemsToken := user.Token
	if rubygemsToken == "" {
		rubygemsToken, err = ui.db.CreateUserToken(username)
		if err != nil {
			log.Error().Err(err).Msg("failed to create user token")
		}
	}
	c.HTML(http.StatusOK, "tokens", gin.H{
		"accessToken":   sessionAuth.(string),
		"authType":      ui.cfg.Auth.Type,
		"rubygemsToken": rubygemsToken,
	})
}
