package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	
	"github.com/gin-gonic/gin"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/models"
	"github.com/gemfast/server/internal/spec"
	"github.com/rs/zerolog/log"
)

func head(c *gin.Context) {
	c.JSON(http.StatusOK, "{}")
}

func createToken(c *gin.Context) {
	user, _ := c.Get(IdentityKey)
	u, _ := user.(*models.User)
	token, err := models.CreateUserToken(u)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate token for user")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"username": u.Username,
	})
}

func listGems(c *gin.Context) {
	gemQuery := c.Query("gem")
	gems, err := models.GetGems(gemQuery)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get gems")
		return
	}
	c.JSON(http.StatusOK, gems)
}

func saveAndReindex(tmpfile *os.File) error {
	s := spec.FromFile(tmpfile.Name())
	fp := fmt.Sprintf("%s/%s-%s.gem", config.Env.GemDir, s.Name, s.Version)
	err := os.Rename(tmpfile.Name(), fp)
	err = models.SetGem(s.Name, s.Version, s.OriginalPlatform)
	go indexer.Get().UpdateIndex()
	return err
}

func fetchGemDependencies(c *gin.Context, gemQuery string) ([]models.Dependency, error) {
	gems := strings.Split(gemQuery, ",")
	var deps []models.Dependency
	for _, gem := range gems {
		existingDeps, err := models.GetDependencies(gem)
		if err != nil {
			log.Trace().Err(err).Str("gem", gem).Msg("failed to fetch dependencies for gem")
			return nil, err
		}
		for _, d := range *existingDeps {
			deps = append(deps, d)
		}
	}
	return deps, nil
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
		log.Error().Err(err).Str("tmpfile", tmpfile.Name()).Msg("failed to save uploaded file")
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
