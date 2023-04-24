package cve

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gemfast/server/internal/config"

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

var AdvisoryDB *cache.Cache

func init() {
	AdvisoryDB = cache.New(24 * time.Hour)
}

func InitRubyAdvisoryDB() error {
	cacheAdvisoryDB(config.Env.RubyAdvisoryDBDir)
	log.Info().Msg("successfully cached ruby advisory DB")
	return nil
}

func cacheAdvisoryDB(path string) {
	var cacheKey string
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		var advisories []GemAdvisory
		if err != nil {
			panic(err)
		}
		if info.IsDir() {
			cacheKey = info.Name()
			return nil
		}
		ga := &GemAdvisory{}
		gemAdvisoryFromFile(path, ga)
		a, found := AdvisoryDB.Get(cacheKey)
		if found {
			advisories = a.([]GemAdvisory)
			advisories = append(advisories, *ga)
		} else {
			advisories = append(advisories, *ga)
		}
		AdvisoryDB.Set(cacheKey, advisories, 0)

		return nil
	})
}

func gemAdvisoryFromFile(path string, ga *GemAdvisory) *GemAdvisory {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, ga)
	if err != nil {
		panic(err)
	}

	return ga
}

func isPatched(gem string, version string) (bool, GemAdvisory, error) {
	var cves []GemAdvisory
	c, found := AdvisoryDB.Get(gem)
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

func isUnaffected(gem string, version string) (bool, GemAdvisory, error) {
	var cves []GemAdvisory
	c, found := AdvisoryDB.Get(gem)
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

func GetCVEs(gem string, version string) []GemAdvisory {
	var cves []GemAdvisory
	patched, cve1, _ := isPatched(gem, version)
	if !patched {
		if !acceptableSeverity(cve1) {
			cves = append(cves, cve1)
		}
		unaffected, cve2, _ := isUnaffected(gem, version)
		if !unaffected {
			if cve2.Cve != cve1.Cve {
				if !acceptableSeverity(cve1) {
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

func acceptableSeverity(cve GemAdvisory) bool {
	severity := severity(cve)
	highestSeverity := strings.ToLower(config.Env.MaxCVESeverity)
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
