package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gemfast/server/internal/db"
	bolt "go.etcd.io/bbolt"
)

type Gem struct {
	Name     string `json:"name"`
	Number   string `json:"number"`
	Platform string `json:"platform"`
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

func GemFromBytes(data []byte) (*[]Gem, error) {
	var p *[]Gem
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func GetGem(name string) ([]Gem, error) {
	var gems []Gem
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		g := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Get([]byte(name))
		gem, _ := GemFromBytes(g)
		gems = *gem
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gems, nil

}

func GetGems() ([][]Gem, error) {
	var gems [][]Gem
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			g, _ := GemFromBytes(v)
			gems = append(gems, *g)
		}
		if gems == nil {
			return fmt.Errorf("no gems found")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gems, nil
}

func GetAllGemversions() []string {
	t := time.Now()
	rfc := t.Format(time.RFC3339)
	arr := []string{fmt.Sprintf("created_at: %s", rfc), "---"}
	m := make(map[string][]string)
	db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			g, _ := GemFromBytes(v)
			gemVersions := *g
			for _, gemVersion := range gemVersions {
				m[gemVersion.Name] = append(m[gemVersion.Name], gemVersion.Number)
			}
		}

		return nil
	})

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sort.Strings(m[k])
		arr = append(arr, k+" "+strings.Join(m[k], ","))
	}
	return arr
}

func GetGemInfo(name string) ([]string, error) {
	var versions []string
	versions = []string{"---"}
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		g := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Get([]byte(name))
		deps, _ := DependenciesFromBytes(g)
		for _, d := range *deps {
			versions = append(versions, d.Line())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(versions)
	return versions, nil
}

func GetAllGemNames() []string {
	var names []string
	names = []string{"---"}
	db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET))
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			names = append(names, string(k))
		}
		return nil
	})
	return names
}

func (d *Dependency) Line() string {
	var l string
	if d.Platform == "" || d.Platform == "ruby" {
		l = d.Number
	} else {
		l = fmt.Sprintf("%s-%s", d.Number, d.Platform)
	}
	l += " "
	arrlen := len(d.Dependencies)
	for i, dep := range d.Dependencies {
		l += dep[0] + ":" + dep[1]
		if i+1 != arrlen {
			l += ","
		}
	}
	l += fmt.Sprintf("|checksum:%s", d.Checksum)
	if d.Ruby != "" && d.Ruby != ">= 0" {
		l += fmt.Sprintf(",ruby:%s", d.Ruby)
	}
	if d.RubyGems != "" && d.RubyGems != ">= 0" {
		l += fmt.Sprintf(",rubygems:%s", d.RubyGems)
	}
	return l
}
