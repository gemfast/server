package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	
	"github.com/gin-gonic/gin"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/marshal"
	"github.com/gemfast/server/internal/models"
	"github.com/gemfast/server/internal/spec"
	"github.com/rs/zerolog/log"
)

func mirroredGemspecRzHandler(c *gin.Context) {
	fileName := c.Param("gemspec.rz")
	fp := filepath.Join(config.Env.Dir, "quick/Marshal.4.8", fileName)
	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(fp)
	  if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to create gem file")
	  }
	  defer out.Close()
	  path, err := url.JoinPath(config.Env.MirrorUpstream, "quick/Marshal.4.8", fileName)
	  if err != nil {
      log.Error().Str("file", fileName).Msg("failed to fetch quick marshal")
      panic(err)
    }
    resp, err := http.Get(path)
    if err != nil {
	    c.String(http.StatusInternalServerError, "Failed to connect to upstream")
	  }
	  defer resp.Body.Close()
	  _, err = io.Copy(out, resp.Body)
	  if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to write gem file")
	  }
	}
	c.FileAttachment(fp, fileName)
}

func mirroredGemHandler(c *gin.Context) {
	fileName := c.Param("gem")
	fp := filepath.Join(config.Env.GemDir, fileName)
	if _, err := os.Stat(fp); errors.Is(err, os.ErrNotExist) {
		out, err := os.Create(fp)
	  if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to create gem file")
	  }
	  defer out.Close()
	  path, err := url.JoinPath(config.Env.MirrorUpstream, "gems", fileName)
	  if err != nil {
      c.String(http.StatusInternalServerError, "Failed to fetch gem file")
      return
    }
	  resp, err := http.Get(path)
	  if err != nil {
	    c.String(http.StatusInternalServerError, "Failed to connect to upstream")
	    return
	  }
	  defer resp.Body.Close()
	  if resp.StatusCode != 200 {
	  	log.Info().Str("upstream", path).Msg("upstream returned a non 200 status code")
	  	c.String(resp.StatusCode, "Failure returned from upstream")
	  	return
	  }
	  _, err = io.Copy(out, resp.Body)
	  if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to write gem file")
	    return
	  }
	  s := spec.FromFile(fp)
		err = models.SetGem(s.Name, s.Version, s.OriginalPlatform)
		if err != nil  {
	    c.String(http.StatusInternalServerError, "Failed to save gem in db")
	    return
	  }
	}
	c.FileAttachment(fp, fileName)
}

func mirroredIndexHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path) 
}

func mirroredInfoHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}

func mirroredVersionsHandler(c *gin.Context) {
	path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
	if err != nil {
		panic(err)
	}
	c.Redirect(http.StatusFound, path)
}

func mirroredDependenciesHandler(c *gin.Context) {
	gemQuery := c.Query("gems")
	log.Trace().Str("gems", gemQuery).Msg("received gems")
	if gemQuery == "" {
		c.Status(http.StatusOK)
		return
	}
	deps, err := fetchGemDependencies(c, gemQuery)
	if err != nil && config.Env.Mirror == "" {
		c.String(http.StatusNotFound, fmt.Sprintf("failed to fetch dependencies for gem: %s", gemQuery))
		return
	} else if err != nil && config.Env.Mirror != "" {
		path, err := url.JoinPath(config.Env.MirrorUpstream, c.FullPath())
		path += "?gems="
		path += gemQuery
		if err != nil {
			panic(err)
		}
		c.Redirect(http.StatusFound, path)
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

func mirroredDependenciesJSONHandler(c *gin.Context) {
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