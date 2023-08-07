package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gin-gonic/gin"
)

type SupermarketHandler struct {
	cfg *config.Config
	db  *db.DB
}

func NewSupermarketHandler(cfg *config.Config, db *db.DB) *SupermarketHandler {
	return &SupermarketHandler{
		cfg: cfg,
		db:  db,
	}
}

func (h *SupermarketHandler) getCookbook(c *gin.Context) {
	cb := c.Params.ByName("cookbook")
	cookbook := &db.Cookbook{Name: cb}
	c.JSON(http.StatusOK, cookbook)
}

func (h *SupermarketHandler) mirroredGetCookbook(c *gin.Context) {
	cb := c.Params.ByName("cookbook")
	path, err := url.JoinPath("https://supermarket.chef.io", c.FullPath())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to join url to create upstream path")
		return
	}
	path = strings.Replace(path, ":cookbook", cb, 1)
	fmt.Println(path)
	c.Redirect(http.StatusFound, path)
}

func (h *SupermarketHandler) createCookbook(c *gin.Context) {
	cookbook := &db.Cookbook{Name: "lol", LatestVersion: "http://supermarket.chef.io/api/v1/cookbooks/apt/versions/2_4_0", Description: "LOL", ExternalURL: "https://github.com/sous-chefs/apt", CreatedAt: time.Now(), UpdatedAt: time.Now(), Deprecated: false}
	c.JSON(http.StatusOK, cookbook)
}
