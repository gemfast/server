package api

import (
	"fmt"
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

// func getGemspecRz(c *gin.Context) {
// 	fileName := c.Param("gemspec.rz")
// 	fp := filepath.Join(config.Env.Dir, "quick/Marshal.4.8", fileName)
// 	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
// 		out, err := os.Create(fp)
// 	  if err != nil  {
// 	    c.String(http.StatusInternalServerError, "Failed to create gem file")
// 	  }
// 	  defer out.Close()
// 	  path, err := url.JoinPath(config.Env.MirrorUpstream, "quick/Marshal.4.8", fileName)
// 	  if err != nil {
//       log.Error().Str("file", fileName).Msg("failed to fetch quick marshal")
//       panic(err)
//     }
//     resp, err := http.Get(path)
//     if err != nil {
// 	    c.String(http.StatusInternalServerError, "Failed to connect to upstream")
// 	  }
// 	  defer resp.Body.Close()
// 	  _, err = io.Copy(out, resp.Body)
// 	  if err != nil  {
// 	    c.String(http.StatusInternalServerError, "Failed to write gem file")
// 	  }
// 	}
// 	c.FileAttachment(fp, fileName)
// }

// func getGem(c *gin.Context) {
// 	fileName := c.Param("gem")
// 	fp := filepath.Join(config.Env.GemDir, fileName)
// 	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
// 		out, err := os.Create(fp)
// 	  if err != nil  {
// 	    c.String(http.StatusInternalServerError, "Failed to create gem file")
// 	  }
// 	  defer out.Close()
// 	  path, err := url.JoinPath(config.Env.MirrorUpstream, "gems", fileName)
// 	  if err != nil {
//       c.String(http.StatusInternalServerError, "Failed to fetch gem file")
//       return
//     }
// 	  resp, err := http.Get(path)
// 	  if err != nil {
// 	    c.String(http.StatusInternalServerError, "Failed to connect to upstream")
// 	    return
// 	  }
// 	  defer resp.Body.Close()
// 	  if resp.StatusCode != 200 {
// 	  	log.Info().Str("upstream", path).Msg("upstream returned a non 200 status code")
// 	  	c.String(resp.StatusCode, "Failure returned from upstream")
// 	  	return
// 	  }
// 	  _, err = io.Copy(out, resp.Body)
// 	  if err != nil  {
// 	    c.String(http.StatusInternalServerError, "Failed to write gem file")
// 	    return
// 	  }
// 	  s := spec.FromFile(fp)
// 		err = models.SetGem(s.Name, s.Version, s.OriginalPlatform)
// 		if err != nil  {
// 	    c.String(http.StatusInternalServerError, "Failed to save gem in db")
// 	    return
// 	  }
// 	}
// 	c.FileAttachment(fp, fileName)
// }

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
		err := checkDb(gem, &deps)
		if err != nil {
			return nil, err
		}
	}
	return deps, nil
}

func checkDb(gem string, deps *[]models.Dependency) (error) {
	existingDeps, err := models.GetDependencies(gem)
	if err != nil {
		log.Trace().Err(err).Str("gem", gem).Msg("failed to fetch dependencies for gem")
		return err
	}
	for _, d := range *existingDeps {
		*deps = append(*deps, d)
	}
	return nil
}

// func getDependencies(c *gin.Context) {
// 	gemQuery := c.Query("gems")
// 	log.Trace().Str("gems", gemQuery).Msg("received gems")
// 	if gemQuery == "" {
// 		c.Status(http.StatusOK)
// 		return
// 	}
// 	deps, err := fetchGemDependencies(c, gemQuery)
// 	if err != nil && config.Env.Mirror == "" {
// 		c.String(http.StatusNotFound, fmt.Sprintf("failed to fetch dependencies for gem: %s", gemQuery))
// 		return
// 	} else if err != nil && config.Env.Mirror != "" {
// 		path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
// 		path += "?gems="
// 		path += gemQuery
// 		if err != nil {
// 			panic(err)
// 		}
// 		c.Redirect(http.StatusFound, path)
// 	}
// 	bundlerDeps, err := marshal.DumpBundlerDeps(deps)
// 	if err != nil {
// 		log.Error().Err(err).Msg("failed to marshal gem dependencies")
// 		c.String(http.StatusInternalServerError, "failed to marshal gem dependencies")
// 		return
// 	}
// 	c.Header("Content-Type", "application/octet-stream; charset=utf-8")
// 	c.Writer.Write(bundlerDeps)
// }

// func getDependenciesJSON(c *gin.Context) {
// 	gemQuery := c.Query("gems")
// 	log.Trace().Str("gems", gemQuery).Msg("received gems")
// 	if gemQuery == "" {
// 		c.Status(http.StatusOK)
// 		return
// 	}
// 	deps, err := fetchGemDependencies(c, gemQuery)
// 	if err != nil {
// 		return
// 	}
// 	c.JSON(http.StatusOK, deps)
// }

