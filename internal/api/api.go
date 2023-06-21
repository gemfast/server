package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/models"
	"github.com/gemfast/server/internal/spec"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func head(c *gin.Context) {
	c.JSON(http.StatusOK, "{}")
}

func listGems(c *gin.Context) {
	gems, err := models.GetGems()
	if err != nil {
		log.Error().Err(err).Msg("failed to get gems")
		c.String(http.StatusInternalServerError, "Failed to get gems")
		return
	}
	c.JSON(http.StatusOK, gems)
}

func getGem(c *gin.Context) {
	name := c.Param("gem")
	gemVersions, err := models.GetGemVersions(name)
	if err != nil {
		log.Error().Err(err).Msg("failed to get gem")
		c.String(http.StatusInternalServerError, "Failed to get gem")
		return
	}
	c.JSON(http.StatusOK, gemVersions)
}

func listUsers(c *gin.Context) {
	users, err := models.GetUsers()
	if err != nil {
		log.Error().Err(err).Msg("failed to get users")
		c.String(http.StatusInternalServerError, "Failed to get users")
		return
	}
	c.JSON(http.StatusOK, users)
}

func getUser(c *gin.Context) {
	username := c.Param("username")
	user, err := models.GetUser(username)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")
		c.String(http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Password = []byte{}
	user.Token = ""
	c.JSON(http.StatusOK, user)
}

func deleteUser(c *gin.Context) {
	username := c.Param("username")
	deleted, err := models.DeleteUser(username)
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

func setUserRole(c *gin.Context) {
	username := c.Param("username")
	role := c.Param("role")
	user, err := models.GetUser(username)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get user")
		return
	}
	user.Role = strings.ToLower(role)
	err = models.UpdateUser(user)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to set user role")
		return
	}
	c.String(http.StatusAccepted, "User role set successfully")
}

func saveAndReindex(tmpfile *os.File) error {
	s, err := spec.FromFile(tmpfile.Name())
	if err != nil {
		log.Error().Err(err).Msg("failed to read spec from tmpfile")
		return err
	}
	var fp string
	if s.OriginalPlatform == "ruby" {
		fp = fmt.Sprintf("%s/%s-%s.gem", config.Cfg.GemDir, s.Name, s.Version)
	} else {
		fp = fmt.Sprintf("%s/%s-%s-%s.gem", config.Cfg.GemDir, s.Name, s.Version, s.OriginalPlatform)
	}
	err = os.Rename(tmpfile.Name(), fp)
	if err != nil {
		log.Error().Err(err).Str("detail", fp).Msg("failed to rename tmpfile")
		return err
	}
	err = indexer.Get().AddGemToIndex(fp)
	if err != nil {
		log.Error().Err(err).Str("detail", s.Name).Msg("failed to add gem to index")
		return err
	}
	return nil
}

func fetchGemVersions(c *gin.Context, gemQuery string) ([]*models.Gem, error) {
	gems := strings.Split(gemQuery, ",")
	var gemVersions []*models.Gem
	for _, gem := range gems {
		gv, err := models.GetGemVersions(gem)
		if err != nil {
			log.Trace().Err(err).Str("detail", gem).Msg("failed to fetch dependencies for gem")
			return nil, err
		}
		for _, g := range gv {
			gemVersions = append(gemVersions, &models.Gem{
				Name:         g.Name,
				Number:       g.Number,
				Dependencies: g.Dependencies,
			})
		}
	}
	return gemVersions, nil
}

func geminaboxUploadGem(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		log.Error().Err(err).Msg("failed to read form file")
		c.String(http.StatusBadRequest, "failed to read form file parameter")
		return
	}
	tmpfile, err := ioutil.TempFile("/tmp", "*.gem")
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
	if err = saveAndReindex(tmpfile); err != nil {
		log.Error().Err(err).Msg("failed to reindex gem")
		c.String(http.StatusInternalServerError, "failed to index gem")
		return
	}
	c.String(http.StatusOK, "uploaded successfully")
}
