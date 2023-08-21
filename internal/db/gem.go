package db

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gemfast/server/internal/spec"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
)

type Gem struct {
	Name         string          `json:"name"`
	Number       string          `json:"number"`
	Platform     string          `json:"platform"`
	Checksum     string          `json:"checksum,omitempty"`
	InfoChecksum string          `json:"info_checksum,omitempty"`
	Ruby         string          `json:"ruby,omitempty"`
	RubyGems     string          `json:"rubygems,omitempty"`
	Dependencies []GemDependency `json:"dependencies,omitempty"`
}

type GemDependency struct {
	Name               string
	Type               string
	VersionConstraints string
}

func GemFromSpec(s *spec.Spec) *Gem {
	return &Gem{
		Name:     s.Name,
		Number:   s.Version,
		Platform: s.OriginalPlatform,
		Checksum: s.Checksum,
		Ruby:     s.Ruby,
		RubyGems: s.RubyGems,
	}
}

func GemFromGemParameter(param string) *Gem {
	var gemName []string
	var gemVersion string
	chunks := strings.Split(param, "-")
	l := len(chunks)
	for i, chunk := range chunks {
		if (i + 1) == l {
			gemVersion = chunk
			break
		}
		gemName = append(gemName, chunk)
	}
	return &Gem{
		Name:   strings.Join(gemName, "-"),
		Number: gemVersion,
	}
}

func GemVersionsFromBytes(data []byte) ([]*Gem, error) {
	var gv []*Gem
	err := json.Unmarshal(data, &gv)
	if err != nil {
		return nil, err
	}
	return gv, nil
}

func (db *DB) SaveGem(source string, g *Gem) error {
	var existing []byte
	db.boltDB.View(func(tx *bolt.Tx) error {
		gemBytes := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source)).Get([]byte(g.Name))
		existing = gemBytes
		return nil
	})
	if existing == nil {
		infoChecksum := CalculateInfoChecksum([]*Gem{g})
		g.InfoChecksum = infoChecksum
		gemBytes, err := json.Marshal([]*Gem{g})
		if err != nil {
			return fmt.Errorf("could not marshal gem to json: %v", err)
		}
		err = db.boltDB.Update(func(tx *bolt.Tx) error {
			err = tx.Bucket([]byte(GemBucket)).Bucket([]byte(source)).Put([]byte(g.Name), gemBytes)
			if err != nil {
				return fmt.Errorf("could not set: %v", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("could not save gem: %v", err)
		}
	} else {
		gemVersions, _ := GemVersionsFromBytes(existing)
		hashed := make(map[string]bool)
		for _, gv := range gemVersions {
			hash := gv.Number + gv.Platform
			hashed[hash] = true
		}
		newHash := g.Number + g.Platform
		if !hashed[newHash] {
			gemArr := append([]*Gem{}, gemVersions...)
			gemArr = append(gemArr, g)
			infoChecksum := CalculateInfoChecksum(gemArr)
			g.InfoChecksum = infoChecksum
			gemVersions = append(gemVersions, g)
			gemBytes, _ := json.Marshal(gemVersions)
			err := db.boltDB.Update(func(tx *bolt.Tx) error {
				err := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source)).Put([]byte(g.Name), gemBytes)
				if err != nil {
					return fmt.Errorf("could not set: %v", err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("could not save gem: %v", err)
			}
		}
	}
	return nil
}

// Create
func (db *DB) SaveGemVersions(source string, specs []*spec.Spec) error {
	for _, s := range specs {
		g := GemFromSpec(s)
		var versionConstraints []string
		var constraint string
		for _, dep := range s.GemMetadata.Dependencies {
			for _, vc := range dep.Requirement.VersionConstraints {
				constraint = fmt.Sprintf("%s %s", vc.Constraint, vc.Version)
				versionConstraints = append(versionConstraints, constraint)
			}
			sort.Strings(versionConstraints)
			g.Dependencies = append(g.Dependencies, GemDependency{
				Name:               dep.Name,
				Type:               dep.Type,
				VersionConstraints: strings.Join(versionConstraints, "&"),
			})
			versionConstraints = []string{}
			constraint = ""
		}
		err := db.SaveGem(source, g)
		if err != nil {
			log.Error().Err(err).Str("detail", g.Name).Msg("failed to save dependencies for gem")
			return err
		}
	}
	return nil
}

// Delete
func (db *DB) DeleteGemVersion(source string, toDelete *Gem) (int, error) {
	count := 0
	gemVersions, err := db.GetGemVersions(source, toDelete.Name)
	if err != nil {
		return count, err
	}
	if toDelete.Platform == "" {
		toDelete.Platform = "ruby"
	}
	for i, gem := range gemVersions {
		if gem.Number == toDelete.Number && gem.Platform == toDelete.Platform {
			gemVersions = slices.Delete(gemVersions, i, i+1)
			count += 1
		}
	}
	gemBytes, _ := json.Marshal(gemVersions)
	_ = db.boltDB.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source)).Put([]byte(toDelete.Name), gemBytes)
		if err != nil {
			return fmt.Errorf("could not set: %v", err)
		}
		return nil
	})
	return count, nil
}

// Read
func (db *DB) GetGemVersions(source, name string) ([]*Gem, error) {
	var gems []*Gem
	err := db.boltDB.View(func(tx *bolt.Tx) error {
		g := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source)).Get([]byte(name))
		gem, _ := GemVersionsFromBytes(g)
		gems = gem
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gems, nil

}

func (db *DB) GetGems(source string) ([][]*Gem, error) {
	var allGems [][]*Gem
	err := db.boltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			g, _ := GemVersionsFromBytes(v)
			allGems = append(allGems, g)
		}
		if len(allGems) == 0 {
			log.Debug().Msg("no gems found")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return allGems, nil
}

func (db *DB) GetAllGemversions(source string) ([]string, error) {
	t := time.Now()
	rfc := t.Format(time.RFC3339)
	arr := []string{fmt.Sprintf("created_at: %s", rfc), "---"}
	m := make(map[string][]string)
	err := db.boltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			gemVersions, err := GemVersionsFromBytes(v)
			if err != nil {
				return err
			}
			l := len(gemVersions)
			for i, gv := range gemVersions {
				if i == l-1 {
					m[gv.Name] = append(m[gv.Name], gv.Number+" "+gv.InfoChecksum)
				} else {
					m[gv.Name] = append(m[gv.Name], gv.Number)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sort.Strings(m[k])
		arr = append(arr, k+" "+strings.Join(m[k], ","))
	}
	return arr, nil
}

func (db *DB) GetGemInfo(source, name string) (string, error) {
	var gemVersions []*Gem
	err := db.boltDB.View(func(tx *bolt.Tx) error {
		g := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source)).Get([]byte(name))
		gv, err := GemVersionsFromBytes(g)
		gemVersions = gv
		return err
	})
	if err != nil {
		return "", err
	}
	return CompactIndexInfo(gemVersions), nil
}

func (db *DB) GetAllGemNames(source string) []string {
	var names []string
	names = []string{"---"}
	db.boltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source))
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			names = append(names, string(k))
		}
		return nil
	})
	return names
}

func CompactIndexInfo(gems []*Gem) string {
	var l string
	versions := []string{"---"}
	for _, g := range gems {
		if g.Platform == "" || g.Platform == "ruby" {
			l = g.Number
		} else {
			l = fmt.Sprintf("%s-%s", g.Number, g.Platform)
		}
		l += " "
		for _, dep := range g.Dependencies {
			if dep.Type == ":runtime" {
				l += dep.Name + ":" + dep.VersionConstraints + ","
			}
		}
		l = strings.TrimSuffix(l, ",")
		l += fmt.Sprintf("|checksum:%s", g.Checksum)
		if g.Ruby != "" && g.Ruby != ">= 0" {
			l += fmt.Sprintf(",ruby:%s", g.Ruby)
		}
		if g.RubyGems != "" && g.RubyGems != ">= 0" {
			l += fmt.Sprintf(",rubygems:%s", g.RubyGems)
		}
		versions = append(versions, l)
	}
	sort.Strings(versions)
	return strings.Join(versions, "\n") + "\n"
}

func CalculateInfoChecksum(gems []*Gem) string {
	info := CompactIndexInfo(gems)
	md5 := md5.Sum([]byte(info))
	return hex.EncodeToString(md5[:])
}

func (db *DB) SearchGems(source, name string) []string {
	var names []string
	db.boltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source))
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if strings.Contains(string(k), name) {
				names = append(names, string(k))
			}
		}
		return nil
	})
	return names
}

func (db *DB) PrefixScanGems(source, prefix string) []string {
	var names []string
	bytePrefix := []byte(prefix)
	db.boltDB.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(GemBucket)).Bucket([]byte(source)).Cursor()
		for k, _ := c.Seek(bytePrefix); k != nil && bytes.HasPrefix(k, bytePrefix); k, _ = c.Next() {
			names = append(names, string(k))
		}
		return nil
	})
	return names
}
