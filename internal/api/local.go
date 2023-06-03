package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/indexer"
	"github.com/gemfast/server/internal/models"
	"github.com/gemfast/server/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func localGemspecRzHandler(c *gin.Context) {
	fileName := c.Param("gemspec.rz")
	fp := filepath.Join(config.Cfg.Dir, "quick/Marshal.4.8", fileName)
	c.FileAttachment(fp, fileName)
}

func localGemHandler(c *gin.Context) {
	fileName := c.Param("gem")
	fp := filepath.Join(config.Cfg.GemDir, fileName)
	c.FileAttachment(fp, fileName)
}

func localIndexHandler(c *gin.Context) {
	s := strings.Split(c.FullPath(), "/")
	l := len(s)
	c.File(filepath.Join(config.Cfg.Dir, s[l-1]))
}

func localDependenciesHandler(c *gin.Context) {
	// gemQuery := c.Query("gems")
	// log.Trace().Str("gems", gemQuery).Msg("received gems")
	// if gemQuery == "" {
	// 	c.Status(http.StatusOK)
	// 	return
	// }
	// deps, err := fetchGemDependencies(c, gemQuery)
	// if err != nil && config.Cfg.MirrorEnabled != "false" {
	// 	c.String(http.StatusNotFound, fmt.Sprintf("failed to fetch dependencies for gem: %s", gemQuery))
	// 	return
	// } else if err != nil && config.Cfg.MirrorEnabled != "false" {
	// 	path, err := url.JoinPath(config.Cfg.MirrorUpstream, c.FullPath())
	// 	path += "?gems="
	// 	path += gemQuery
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	c.Redirect(http.StatusFound, path)
	// }
	// bundlerDeps, err := marshal.DumpBundlerDeps(deps)
	// if err != nil {
	// 	log.Error().Err(err).Msg("failed to marshal gem dependencies")
	// 	c.String(http.StatusInternalServerError, fmt.Sprintf("failed to marshal gem dependencies: %v", err))
	// 	return
	// }
	// c.Header("Content-Type", "application/octet-stream; charset=utf-8")
	// c.Writer.Write(bundlerDeps)
}

func localDependenciesJSONHandler(c *gin.Context) {
	// gemQuery := c.Query("gems")
	// log.Trace().Str("gems", gemQuery).Msg("received gems")
	// if gemQuery == "" {
	// 	c.Status(http.StatusOK)
	// 	return
	// }
	// deps, err := fetchGemDependencies(c, gemQuery)
	// if err != nil {
	// 	return
	// }
	c.JSON(http.StatusOK, nil)
}

func localUploadGemHandler(c *gin.Context) {
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	tmpfile, err := ioutil.TempFile("/tmp", "*.gem")
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmp file")
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to index gem: %v", err))
		return
	}
	defer os.Remove(tmpfile.Name())

	err = os.WriteFile(tmpfile.Name(), bodyBytes, 0644)
	if err != nil {
		log.Error().Err(err).Str("tmpfile", tmpfile.Name()).Msg("failed to save uploaded file")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to index gem: %v", err))
		return
	}
	if err = saveAndReindex(tmpfile); err != nil {
		log.Error().Err(err).Msg("failed to reindex gem")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to index gem: %v", err))
		return
	}
	c.String(http.StatusOK, "uploaded successfully")
}

func localYankHandler(c *gin.Context) {
	g := c.Query("gem")
	v := c.Query("version")
	p := c.Query("platform")
	if g == "" || v == "" {
		c.String(http.StatusBadRequest, "must provide both gem and version query parameters")
		return
	}
	err := indexer.Get().RemoveGemFromIndex(g, v, p)
	if err != nil {
		log.Error().Err(err).Msg("failed to yank gem from index")
		c.String(http.StatusInternalServerError, fmt.Sprintf("server failed to yank gem from index: %v", err))
		return
	}
	fileName := g + "-" + v + ".gem"
	fp := filepath.Join(config.Cfg.GemDir, fileName)
	err = utils.RemoveFileIfExists(fp)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete gem file system")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete gem from file system: %v", err))
		return
	}
	fileName = fileName + "spec.rz"
	fp = filepath.Join(config.Cfg.Dir, "quick/Marshal.4.8", fileName)
	err = utils.RemoveFileIfExists(fp)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete gemspec.rz from file system")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to delete gem file system: %v", err))
		return
	}
	num, err := models.DeleteGemVersion(&models.Gem{Name: g, Number: v, Platform: p})
	if err != nil {
		log.Error().Err(err).Msg("failed to yank gem")
		c.String(http.StatusInternalServerError, fmt.Sprintf("server failed to yank gem: %v", err))
		return
	}
	if num == 0 {
		c.String(http.StatusNotFound, "no gem matching %s-%s-%s found", g, v, p)
		return
	}
	c.String(http.StatusOK, "successfully yanked")
}

func localVersionsHandler(c *gin.Context) {
	versions := models.GetAllGemversions()
	// if err != nil {
	// 	log.Error().Err(err).Msg("failed to get gem versions")
	// 	c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get gem versions: %v", err))
	// 	return
	// }
	c.String(http.StatusOK, strings.Join(versions, "\n"))
}

func localNamesHandler(c *gin.Context) {
	names := models.GetAllGemNames()
	// if err != nil {
	// 	log.Error().Err(err).Msg("failed to get gem names")
	// 	c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get gem names: %v", err))
	// 	return
	// }
	c.String(http.StatusOK, (strings.Join(names, "\n") + "\n"))
}

func localInfoHandler(c *gin.Context) {
	gem := c.Param("gem")
	if gem == "" {
		c.String(http.StatusBadRequest, "must provide gem name")
		return
	}
	info, err := models.GetGemInfo(gem)
	if err != nil {
		log.Error().Err(err).Msg("failed to get gem info")
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to get gem info: %v", err))
		return
	}
	c.String(http.StatusOK, info+"\n")
}
