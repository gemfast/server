package models

import (
	"encoding/json"
	// "errors"
	"fmt"
	"sync"

	"github.com/gemfast/server/internal/db"
	bolt "go.etcd.io/bbolt"
)

var glock sync.Mutex

type Gem struct {
	Name     string
	Version  string
	Platform string
}

func GemFromBytes(data []byte) (*[]Gem, error) {
	var p *[]Gem
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// func GetGems() (*[]Gem, error) {
// 	var existing []byte
// 	err := db.BoltDB.View(func(tx *bolt.Tx) error {
// 		gems := tx.Bucket([]byte(db.GEM_BUCKET)).Get([]byte(name))
// 		if gems == nil {
// 			return errors.New("no gems found")
// 		}
// 		existing = gems
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return GemFromBytes(existing)
// }

func SetGem(name string, version string, platform string) error {
	glock.Lock()
	defer glock.Unlock()
	var existing []byte
	gem := Gem{
		Name: name,
		Version:   version,
		Platform:  platform, 
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
