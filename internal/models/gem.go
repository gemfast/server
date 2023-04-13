package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/exp/slices"

	"github.com/gemfast/server/internal/db"
	bolt "go.etcd.io/bbolt"
)

type Gem struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Platform string `json:platform`
}

func GemFromBytes(data []byte) (*[]Gem, error) {
	var p *[]Gem
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func GetGems(name string) ([][]Gem, error) {
	var gems [][]Gem
	if name == "" {
		err := db.BoltDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(db.GEM_BUCKET))
			c := b.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				g, _ := GemFromBytes(v)
				gems = append(gems, *g)
			}
			if gems == nil {
				return errors.New("no gems found")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return gems, nil
	} else {
		err := db.BoltDB.View(func(tx *bolt.Tx) error {
			v := tx.Bucket([]byte(db.GEM_BUCKET)).Get([]byte(name))
			g, err := GemFromBytes(v)
			if err != nil {
				return err
			}
			if g == nil {
				return fmt.Errorf("no gem versions for gem %s", name)
			}
			gems = append(gems, *g)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return gems, nil
	}
}

func SetGem(name string, version string, platform string) error {
	var existing []byte
	gem := Gem{
		Name:     name,
		Version:  version,
		Platform: platform,
	}
	db.BoltDB.View(func(tx *bolt.Tx) error {
		existing = tx.Bucket([]byte(db.GEM_BUCKET)).Get([]byte(gem.Name))
		return nil
	})
	if existing == nil {
		gemBytes, err := json.Marshal([]Gem{gem})
		if err != nil {
			return fmt.Errorf("could not marshal gem to json: %v", err)
		}
		err = db.BoltDB.Update(func(tx *bolt.Tx) error {
			err = tx.Bucket([]byte(db.GEM_BUCKET)).Put([]byte(gem.Name), gemBytes)
			if err != nil {
				return fmt.Errorf("could not set: %v", err)
			}
			return nil
		})
	} else {
		gems, _ := GemFromBytes(existing)
		hashed := make(map[string]bool)
		for _, g := range *gems {
			hash := g.Version + g.Platform
			hashed[hash] = true
		}
		newHash := version + platform
		if !hashed[newHash] {
			*gems = append(*gems, gem)
			gemBytes, _ := json.Marshal(*gems)
			_ = db.BoltDB.Update(func(tx *bolt.Tx) error {
				err := tx.Bucket([]byte(db.GEM_BUCKET)).Put([]byte(name), gemBytes)
				if err != nil {
					return fmt.Errorf("could not set: %v", err)
				}
				return nil
			})
		}
	}
	return nil
}

func DeleteGem(name string, version string, platform string) (int, error) {
	var updatedGems []Gem
	count := 0
	if platform == "" {
		platform = "ruby"
	}
	gems, err := GetGems(name)
	if err != nil {
		return 0, err
	}

	for i, g := range gems[0] {
		if version == g.Version && platform == g.Platform {
			updatedGems = slices.Delete(gems[0], i, i+1)
			count++
		}
	}
	if len(updatedGems) == 0 {
		err = db.BoltDB.Update(func(tx *bolt.Tx) error {
			err := tx.Bucket([]byte(db.GEM_BUCKET)).Delete([]byte(name))
			if err != nil {
				return fmt.Errorf("could not delete: %v", err)
			}
			return nil
		})
	} else {
		gemBytes, _ := json.Marshal(updatedGems)
		err = db.BoltDB.Update(func(tx *bolt.Tx) error {
			err := tx.Bucket([]byte(db.GEM_BUCKET)).Put([]byte(name), gemBytes)
			if err != nil {
				return fmt.Errorf("could not set: %v", err)
			}
			return nil
		})
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}
