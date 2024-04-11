package cve

import (
	"fmt"
	"time"

	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gemfast/server/internal/config"
	git "github.com/go-git/go-git/v5"

	"github.com/akyoto/cache"
	ggv "github.com/aquasecurity/go-gem-version"
	"github.com/rs/zerolog/log"
)

type GemAdvisory struct {
	Gem                string   `yaml:"gem"`
	Cve                string   `yaml:"cve"`
	Date               string   `yaml:"date"`
	URL                string   `yaml:"url"`
	Title              string   `yaml:"title"`
	Description        string   `yaml:"description"`
	CvssV2             float64  `yaml:"cvss_v2"`
	CvssV3             float64  `yaml:"cvss_v3"`
	PatchedVersions    []string `yaml:"patched_versions"`
	UnaffectedVersions []string `yaml:"unaffected_versions"`
	Related            struct {
		Cve []string `yaml:"cve"`
		URL []string `yaml:"url"`
	} `yaml:"related"`
}

type GemAdvisoryDB struct {
	db  *cache.Cache
	cfg *config.Config
}

func NewGemAdvisoryDB(cfg *config.Config) *GemAdvisoryDB {
	if !cfg.CVE.Enabled {
		log.Trace().Msg("ruby advisory db disabled")
	}
	advisoryDB := cache.New(24 * time.Hour)
	return &GemAdvisoryDB{
		db:  advisoryDB,
		cfg: cfg,
	}
}

func (g *GemAdvisoryDB) Refresh() error {
	err := g.updateAdvisoryRepo()
	if err != nil {
		log.Warn().Err(err).Msg("failed to update ruby-advisory-db")
	}
	g.db.Close()
	err = g.cacheAdvisoryDB("/gems")
	if err != nil {
		log.Error().Err(err).Msg("failed to cache github.com/rubysec/ruby-advisory-db")
		return fmt.Errorf("failed to cache github.com/rubysec/ruby-advisory-db: %w", err)
	}
	log.Info().Msg("successfully cached github.com/rubysec/ruby-advisory-db")
	return nil
}

func (g *GemAdvisoryDB) updateAdvisoryRepo() error {
	raDB := g.cfg.CVE.RubyAdvisoryDBDir
	if _, err := os.Stat(raDB); os.IsNotExist(err) {
		_, err := git.PlainClone(raDB, false, &git.CloneOptions{
			URL: "https://github.com/rubysec/ruby-advisory-db.git",
			// Depth: 1,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to clone github.com/rubysec/ruby-advisory-db")
			return fmt.Errorf("failed to clone github.com/rubysec/ruby-advisory-db: %w", err)
		}
	} else {
		r, err := git.PlainOpen(raDB)
		if err != nil {
			return fmt.Errorf("failed to open github.com/rubysec/ruby-advisory-db git directory: %w", err)
		}
		w, err := r.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get github.com/rubysec/ruby-advisory-db worktree: %w", err)
		}
		log.Info().Msg("updating github.com/rubysec/ruby-advisory-db")
		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil && err.Error() == "already up-to-date" {
			log.Info().Msg("ruby-advisory-db is already up to date")
		} else if err != nil {
			log.Error().Err(err).Msg("failed to update github.com/rubysec/ruby-advisory-db")
		}

	}
	return nil
}

func (g *GemAdvisoryDB) cacheAdvisoryDB(suffix string) error {
	var cacheKey string
	path := g.cfg.CVE.RubyAdvisoryDBDir + suffix
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("failed to stat github.com/rubysec/ruby-advisory-db: %w", err)
	}
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		var advisories []GemAdvisory
		if err != nil {
			return err
		}
		if info.IsDir() {
			cacheKey = info.Name()
			return nil
		}
		ga, err := gemAdvisoryFromFile(path)
		if err != nil {
			return err
		}
		a, found := g.db.Get(cacheKey)
		if found {
			advisories = a.([]GemAdvisory)
			advisories = append(advisories, *ga)
		} else {
			advisories = append(advisories, *ga)
		}
		g.db.Set(cacheKey, advisories, 0)

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to walk github.com/rubysec/ruby-advisory-db")
		return fmt.Errorf("failed to walk github.com/rubysec/ruby-advisory-db: %w", err)
	}
	return nil
}

func gemAdvisoryFromFile(path string) (*GemAdvisory, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	ga := &GemAdvisory{}
	err = yaml.Unmarshal(yamlFile, ga)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml file %s: %w", path, err)
	}

	return ga, nil
}

func (g *GemAdvisoryDB) isPatched(gem string, version string) (bool, GemAdvisory, error) {
	var cves []GemAdvisory
	c, found := g.db.Get(gem)
	if !found {
		return true, GemAdvisory{}, nil
	}

	gv, err := ggv.NewVersion(version)
	if err != nil {
		return false, GemAdvisory{}, err
	}
	cves = c.([]GemAdvisory)

	for _, cve := range cves {
		if !isPatchedVersion(gv, cve) {
			return false, cve, nil
		}
	}
	return true, GemAdvisory{}, nil
}

func isPatchedVersion(version ggv.Version, cve GemAdvisory) bool {
	for _, pv := range cve.PatchedVersions {
		c, _ := ggv.NewConstraints(pv)
		if c.Check(version) {
			return true
		}
	}
	return false
}

func (g *GemAdvisoryDB) isUnaffected(gem string, version string) (bool, GemAdvisory, error) {
	var cves []GemAdvisory
	c, found := g.db.Get(gem)
	if !found {
		return true, GemAdvisory{}, nil
	}

	gv, err := ggv.NewVersion(version)
	if err != nil {
		return false, GemAdvisory{}, err
	}
	cves = c.([]GemAdvisory)

	for _, cve := range cves {
		if !isUnaffectedVersion(gv, cve) {
			return false, cve, nil
		}
	}
	return true, GemAdvisory{}, nil
}

func isUnaffectedVersion(version ggv.Version, cve GemAdvisory) bool {
	for _, pv := range cve.UnaffectedVersions {
		c, _ := ggv.NewConstraints(pv)
		if c.Check(version) {
			return true
		}
	}
	return false
}

func (g *GemAdvisoryDB) GetCVEs(gem string, version string) []GemAdvisory {
	var cves []GemAdvisory
	patched, cve1, _ := g.isPatched(gem, version)
	if !patched {
		if !g.acceptableSeverity(cve1) {
			cves = append(cves, cve1)
		}
		unaffected, cve2, _ := g.isUnaffected(gem, version)
		if !unaffected {
			if cve2.Cve != cve1.Cve {
				if !g.acceptableSeverity(cve1) {
					cves = append(cves, cve2)
				}
			}
			return cves
		}
	}
	return cves
}

func severity(cve GemAdvisory) string {
	if cve.CvssV3 != 0 {
		if cve.CvssV3 == 0.0 {
			return "none"
		} else if cve.CvssV3 >= 0.1 && cve.CvssV3 <= 3.9 {
			return "low"
		} else if cve.CvssV3 >= 4.0 && cve.CvssV3 <= 6.9 {
			return "medium"
		} else if cve.CvssV3 >= 7.0 && cve.CvssV3 <= 8.9 {
			return "high"
		} else if cve.CvssV3 >= 9.0 && cve.CvssV3 <= 10.0 {
			return "critical"
		}
	} else if cve.CvssV2 != 0 {
		if cve.CvssV2 == 0.0 && cve.CvssV2 <= 3.9 {
			return "low"
		} else if cve.CvssV2 >= 4.0 && cve.CvssV2 <= 6.9 {
			return "medium"
		} else if cve.CvssV2 >= 7.0 && cve.CvssV2 <= 10.0 {
			return "high"
		}
	}
	return "none"
}

func (g *GemAdvisoryDB) acceptableSeverity(cve GemAdvisory) bool {
	severity := severity(cve)
	highestSeverity := strings.ToLower(g.cfg.CVE.MaxSeverity)
	if severity == "none" || highestSeverity == "critical" {
		return true
	}
	if highestSeverity == "low" {
		return false
	} else if highestSeverity == "medium" {
		return severity == "low" || severity == "medium"
	} else if highestSeverity == "high" {
		return severity == "low" || severity == "medium" || severity == "high"
	}
	return true
}
