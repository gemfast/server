package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gemfast/server/internal/db"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
)

type Dependency struct {
	Name         string
	Number       string
	Platform     string
	Dependencies [][]string
}

func DependenciesFromBytes(data []byte) (*[]Dependency, error) {
	var p *[]Dependency
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func GetDependencies(name string) (*[]Dependency, error) {
	var existing []byte
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		deps := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Get([]byte(name))
		if deps == nil {
			return errors.New("dependencies not found")
		}
		existing = deps
		return nil
	})
	if err != nil {
		return nil, err
	}
	return DependenciesFromBytes(existing)
}

func SetDependencies(name string, newDep Dependency) error {
	var existing []byte
	db.BoltDB.View(func(tx *bolt.Tx) error {
		deps := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Get([]byte(name))
		existing = deps
		return nil
	})
	if existing == nil {
		depBytes, err := json.Marshal([]Dependency{newDep})
		if err != nil {
			return fmt.Errorf("could not marshal dependencies to json: %v", err)
		}
		err = db.BoltDB.Update(func(tx *bolt.Tx) error {
			err = tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Put([]byte(name), depBytes)
			if err != nil {
				return fmt.Errorf("could not set: %v", err)
			}
			return nil
		})
	} else {
		deps, _ := DependenciesFromBytes(existing)
		hashed := make(map[string]bool)
		for _, d := range *deps {
			hash := d.Number + d.Platform
			hashed[hash] = true
		}
		newHash := newDep.Number + newDep.Platform
		if !hashed[newHash] {
			*deps = append(*deps, newDep)
			depBytes, _ := json.Marshal(*deps)
			_ = db.BoltDB.Update(func(tx *bolt.Tx) error {
				err := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Put([]byte(name), depBytes)
				if err != nil {
					return fmt.Errorf("could not set: %v", err)
				}
				return nil
			})
		}
	}
	return nil
}

func DeleteDependencies(name string, version string, platform string) (int, error) {
	var updatedDeps []Dependency
	count := 0
	if platform == "" {
		platform = "ruby"
	}
	deps, err := GetDependencies(name)
	if err != nil {
		return count, err
	}
	for i, d := range *deps {
		if d.Number == version && platform == d.Platform {
			updatedDeps = slices.Delete(*deps, i, i+1)
			count++
		}
	}
	if len(updatedDeps) == 0 {
		err = db.BoltDB.Update(func(tx *bolt.Tx) error {
			err := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Delete([]byte(name))
			if err != nil {
				return fmt.Errorf("could not delete: %v", err)
			}
			return nil
		})
	} else {
		depBytes, _ := json.Marshal(updatedDeps)
		err = db.BoltDB.Update(func(tx *bolt.Tx) error {
			err := tx.Bucket([]byte(db.GEM_DEPENDENCY_BUCKET)).Put([]byte(name), depBytes)
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
