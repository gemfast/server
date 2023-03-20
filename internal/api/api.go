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
	gemQuery := c.Query("gem")
	gems, err := models.GetGems(gemQuery)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get gems")
		return
	}
	c.JSON(http.StatusOK, gems)
}

func saveAndReindex(tmpfile *os.File) error {
	s, err := spec.FromFile(tmpfile.Name())
	if err != nil {
		log.Error().Err(err).Msg("failed to read spec from tmpfile")
		return err
	}
	fp := fmt.Sprintf("%s/%s-%s.gem", config.Env.GemDir, s.Name, s.Version)
	err = os.Rename(tmpfile.Name(), fp)
	if err != nil {
		log.Error().Err(err).Str("gem", fp).Msg("failed to rename tmpfile")
		return err
	}
	err = models.SetGem(s.Name, s.Version, s.OriginalPlatform)
	if err != nil {
		log.Error().Err(err).Str("gem", s.Name).Msg("failed to save gem to db")
		return err
	}
	err = indexer.Get().AddGemToIndex(fp)
	if err != nil {
		log.Error().Err(err).Str("gem", s.Name).Msg("failed to add gem to index")
		return err
	}
	return nil
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
