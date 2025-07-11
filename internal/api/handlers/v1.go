package handlers

import (
	"net/http"
	"strings"

	"github.com/gemfast/server/internal/acl"
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
	ACL        *acl.ACL
}

type errorMessage struct {
	Message string `json:"error_message"`
}

func NewAPIV1Handler(cfg *config.Config, database *db.DB, i *indexer.Indexer, f *filter.RegexFilter, advisoryDB *cve.GemAdvisoryDB, acl *acl.ACL) *APIV1Handler {
	return &APIV1Handler{
		cfg:        cfg,
		db:         database,
		indexer:    i,
		Filter:     f,
		advisoryDB: advisoryDB,
		ACL:        acl,
	}
}

func (h *APIV1Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, "{}")
}

func (h *APIV1Handler) GetAuthMode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"auth": h.cfg.Auth.Type})
}

func (h *APIV1Handler) ListGems(c *gin.Context) {
	source := c.Param("source")
	gems, err := h.db.GetGems(source)
	if err != nil {
		log.Error().Err(err).Msg("failed to get gems")
		c.JSON(http.StatusInternalServerError, &errorMessage{"Failed to get gems"})
		return
	}
	if len(gems) == 0 {
		c.JSON(http.StatusOK, []string{})
		return
	}
	for _, gemVersions := range gems {
		for _, gv := range gemVersions {
			gv.Minimal()
		}
	}
	c.JSON(http.StatusOK, gems)
}

func (h *APIV1Handler) SearchGems(c *gin.Context) {
	name := c.Param("name")
	source := c.Param("source")
	matches := h.db.SearchGems(source, name)
	if len(matches) == 0 {
		c.JSON(http.StatusOK, []string{})
		return
	}
	c.JSON(http.StatusOK, matches)
}

func (h *APIV1Handler) PrefixScanGems(c *gin.Context) {
	prefix := c.Param("prefix")
	source := c.Param("source")
	matches := h.db.PrefixScanGems(source, prefix)
	if len(matches) == 0 {
		c.JSON(http.StatusOK, []string{})
		return
	}
	c.JSON(http.StatusOK, matches)
}

func (h *APIV1Handler) GetGem(c *gin.Context) {
	name := c.Param("gem")
	source := c.Param("source")
	gemVersions, err := h.db.GetGemVersions(source, name)
	if err != nil {
		log.Error().Err(err).Msg("failed to get gem")
		c.JSON(http.StatusInternalServerError, "failed to get gem")
		return
	}
	c.JSON(http.StatusOK, gemVersions)
}

func (h *APIV1Handler) ListUsers(c *gin.Context) {
	users, err := h.db.GetUsers()
	if err != nil {
		log.Error().Err(err).Msg("failed to get users")
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to get users"})
		return
	}
	if len(users) == 0 {
		c.JSON(http.StatusOK, []string{})
		return
	}
	for _, u := range users {
		u.HidePassword()
	}
	c.JSON(http.StatusOK, users)
}

func (h *APIV1Handler) GetUser(c *gin.Context) {
	username := c.Param("username")
	user, err := h.db.GetUser(username)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to get user"})
		return
	}
	user.HidePassword()
	c.JSON(http.StatusOK, user)
}

func (h *APIV1Handler) DeleteUser(c *gin.Context) {
	username := c.Param("username")
	deleted, err := h.db.DeleteUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to delete user"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, errorMessage{"user not found"})
		return
	}
	c.JSON(http.StatusAccepted, errorMessage{"user deleted successfully"})
}

func (h *APIV1Handler) SetUserRole(c *gin.Context) {
	username := c.Param("username")
	role := c.Param("role")
	user, err := h.db.GetUser(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to get user"})
		return
	}
	user.Role = strings.ToLower(role)
	err = h.db.UpdateUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to set user role"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "user role set successfully"})
}

func (h *APIV1Handler) Backup(c *gin.Context) {
	err := h.db.Backup(c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to backup database"})
		return
	}
}

func (h *APIV1Handler) DBStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.db.Stats())
}

func (h *APIV1Handler) BucketStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.db.BucketStats())
}

func (h *APIV1Handler) ListPolicies(c *gin.Context) {
	policies, err := h.ACL.Casbin.GetPolicy()
	if err != nil {
		log.Error().Err(err).Msg("failed to get policies")
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to get policies"})
		return
	}
	p := db.SliceToACLPolicies(policies)
	c.JSON(http.StatusOK, gin.H{"policies": p})
}

func (h *APIV1Handler) AddPolicy(c *gin.Context) {
	var aclPolicy db.ACLPolicy
	if err := c.ShouldBindJSON(&aclPolicy); err != nil {
		c.JSON(http.StatusBadRequest, errorMessage{"invalid request body"})
		return
	}
	if aclPolicy.Role == "admin" {
		log.Error().Msg("cannot add admin policy")
		c.JSON(http.StatusForbidden, errorMessage{"cannot modify admin policy"})
		return
	}
	if err := aclPolicy.Validate(); err != nil {
		log.Error().Err(err).Msg("invalid policy request")
		c.JSON(http.StatusBadRequest, errorMessage{"invalid policy request"})
		return
	}
	added, err := h.ACL.Casbin.AddPolicy(aclPolicy.Role, aclPolicy.Resource, aclPolicy.Action)
	if err != nil {
		log.Error().Err(err).Msg("failed to add policy")
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to add policy"})
		return
	}
	if !added {
		c.JSON(http.StatusConflict, errorMessage{"policy already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "policy added"})
}

func (h *APIV1Handler) RemovePolicy(c *gin.Context) {
	var aclPolicy db.ACLPolicy
	if err := c.ShouldBindJSON(&aclPolicy); err != nil {
		log.Error().Err(err).Msg("failed to bind request body")
		c.JSON(http.StatusBadRequest, errorMessage{"invalid request body"})
		return
	}
	if aclPolicy.Role == "admin" {
		log.Error().Msg("cannot remove admin policy")
		c.JSON(http.StatusForbidden, errorMessage{"cannot modify admin policy"})
		return
	}
	removed, err := h.ACL.Casbin.RemovePolicy(aclPolicy.Role, aclPolicy.Resource, aclPolicy.Action)
	if err != nil {
		log.Error().Err(err).Msg("failed to remove policy")
		c.JSON(http.StatusInternalServerError, errorMessage{"failed to remove policy"})
		return
	}
	if !removed {
		c.JSON(http.StatusNotFound, errorMessage{"policy not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "policy removed"})
}
