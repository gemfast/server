package api

import (
	"net/http"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/cve"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/filter"
	"github.com/gemfast/server/internal/indexer"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type APIV1Handler struct {
	cfg        *config.Config
	db         *db.DB
	indexer    *indexer.Indexer
	Filter     *filter.RegexFilter
	advisoryDB *cve.GemAdvisoryDB
}

func NewAPIV1Handler(cfg *config.Config, database *db.DB, i *indexer.Indexer, f *filter.RegexFilter, advisoryDB *cve.GemAdvisoryDB) *APIV1Handler {
	return &APIV1Handler{
		cfg:        cfg,
		db:         database,
		indexer:    i,
		Filter:     f,
		advisoryDB: advisoryDB,
	}
}

func (h *APIV1Handler) health(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><body style=\"background-color: green\"></body></html>"))
}

func (h *APIV1Handler) authMode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"auth": h.cfg.Auth.Type})
}

func (h *APIV1Handler) listGems(c *gin.Context) {
	gems, err := h.db.GetGems()
	if err != nil {
		log.Error().Err(err).Msg("failed to get gems")
		c.String(http.StatusInternalServerError, "Failed to get gems")
		return
	}
	c.JSON(http.StatusOK, gems)
}

func (h *APIV1Handler) getGem(c *gin.Context) {
	name := c.Param("gem")
	gemVersions, err := h.db.GetGemVersions(name)
	if err != nil {
		log.Error().Err(err).Msg("failed to get gem")
		c.String(http.StatusInternalServerError, "Failed to get gem")
		return
	}
	c.JSON(http.StatusOK, gemVersions)
}

func (h *APIV1Handler) listUsers(c *gin.Context) {
	users, err := h.db.GetUsers()
	if err != nil {
		log.Error().Err(err).Msg("failed to get users")
		c.String(http.StatusInternalServerError, "Failed to get users")
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *APIV1Handler) getUser(c *gin.Context) {
	username := c.Param("username")
	user, err := h.db.GetUser(username)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")
		c.String(http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Password = []byte{}
	user.Token = ""
	c.JSON(http.StatusOK, user)
}

func (h *APIV1Handler) deleteUser(c *gin.Context) {
	username := c.Param("username")
	deleted, err := h.db.DeleteUser(username)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete user")
		return
	}
	if !deleted {
		c.String(http.StatusNotFound, "User not found")
		return
	}
	c.String(http.StatusAccepted, "User deleted successfully")
}

func (h *APIV1Handler) setUserRole(c *gin.Context) {
	username := c.Param("username")
	role := c.Param("role")
	user, err := h.db.GetUser(username)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Role = strings.ToLower(role)
	err = h.db.UpdateUser(user)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to set user role")
		return
	}
	c.String(http.StatusAccepted, "User role set successfully")
}

func (h *APIV1Handler) backup(c *gin.Context) {
	err := h.db.Backup(c.Writer)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to backup database")
		return
	}
}
