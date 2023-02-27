package api

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	// jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/marshal"
	"github.com/gemfast/server/internal/models"
	"github.com/gemfast/server/internal/spec"
	"github.com/rs/zerolog/log"
)

func head(c *gin.Context) {}

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

func getGemspecRz(c *gin.Context) {
	fileName := c.Param("gemspec.rz")
	filePath := fmt.Sprintf("%s/quick/Marshal.4.8/%s", config.Env.Dir, fileName)
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(filePath)
	  if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to create gem file")
	  }
	  defer out.Close()
	  client := &http.Client{}
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/quick/Marshal.4.8%s", config.Env.MirrorUpstream, fileName), nil)
    if err != nil {
      panic(err)
    }
	  resp, err := client.Do(req)
	  if err != nil {
	    c.String(http.StatusInternalServerError, "Failed to connect to upstream")
	  }
	  defer resp.Body.Close()
	  _, err = io.Copy(out, resp.Body)
	  if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to write gem file")
	  }

	}
	c.FileAttachment(filePath, fileName)
}

func getGem(c *gin.Context) {
	fileName := c.Param("gem")
	filePath := fmt.Sprintf("%s/%s", config.Env.GemDir, fileName)
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(filePath)
	  if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to create gem file")
	  }
	  defer out.Close()
	  resp, err := http.Get(fmt.Sprintf("%s/gems%s", config.Env.MirrorUpstream, fileName))
	  if err != nil {
	    c.String(http.StatusInternalServerError, "Failed to connect to upstream")
	  }
	  defer resp.Body.Close()
	  _, err = io.Copy(out, resp.Body)
	  if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to write gem file")
	  }

	} 
	c.FileAttachment(filePath, fileName)
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
	filePath := fmt.Sprintf("%s/%s-%s.gem", config.Env.GemDir, s.Name, s.Version)
	err := os.Rename(tmpfile.Name(), filePath)
	err = models.SetGem(s.Name, s.Version, s.OriginalPlatform)
	go indexer.Get().UpdateIndex()
	return err
}

func uploadGem(c *gin.Context) {
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	tmpfile, err := ioutil.TempFile("/tmp", "*.gem")
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmp file")
		c.String(http.StatusInternalServerError, "Failed to index gem")
		return
	}
	defer os.Remove(tmpfile.Name())

	err = os.WriteFile(tmpfile.Name(), bodyBytes, 0644)
	if err != nil {
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

func fetchGemDependencies(c *gin.Context, gemQuery string) ([]models.Dependency, error) {
	gems := strings.Split(gemQuery, ",")
	var deps []models.Dependency
	for _, gem := range gems {
		existingDeps, err := models.GetDependencies(gem)
		if err != nil {
			log.Trace().Err(err).Str("gem", gem).Msg("failed to fetch dependencies for gem")
			c.String(http.StatusNotFound, fmt.Sprintf("failed to fetch dependencies for gem: %s", gem))
			return nil, err
		}
		for _, d := range *existingDeps {
			deps = append(deps, d)
		}
	}
	return deps, nil
}

func getDependencies(c *gin.Context) {
	gemQuery := c.Query("gems")
	log.Trace().Str("gems", gemQuery).Msg("received gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	deps, err := fetchGemDependencies(c, gemQuery)
	if err != nil {
		return
	}
	bundlerDeps, err := marshal.DumpBundlerDeps(deps)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal gem dependencies")
		c.String(http.StatusInternalServerError, "failed to marshal gem dependencies")
		return
	}
	c.Header("Content-Type", "application/octet-stream; charset=utf-8")
	c.Writer.Write(bundlerDeps)
}

func getDependenciesJSON(c *gin.Context) {
	gemQuery := c.Query("gems")
	log.Trace().Str("gems", gemQuery).Msg("received gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	deps, err := fetchGemDependencies(c, gemQuery)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, deps)
}
