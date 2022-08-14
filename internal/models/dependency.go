package models

import (
	"encoding/json"
	"fmt"

	"github.com/gscho/gemfast/internal/db"
	bolt "go.etcd.io/bbolt"
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
	db.DB.View(func(tx *bolt.Tx) error {
		deps := tx.Bucket([]byte(db.ROOT_BUCKET)).Get([]byte(name))
		existing = deps
		fmt.Println(string(deps))
		return nil
	})
	return DependenciesFromBytes(existing)
}

func SetDependencies(name string, newDep Dependency) error {
	var existing []byte
	db.DB.View(func(tx *bolt.Tx) error {
		deps := tx.Bucket([]byte(db.ROOT_BUCKET)).Get([]byte(name))
		existing = deps
		return nil
	})
	if existing == nil {
		depBytes, err := json.Marshal([]Dependency{newDep})
		if err != nil {
			return fmt.Errorf("could not marshal config json: %v", err)
		}
		err = db.DB.Update(func(tx *bolt.Tx) error {
			err = tx.Bucket([]byte(db.ROOT_BUCKET)).Put([]byte(name), depBytes)
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
			_ = db.DB.Update(func(tx *bolt.Tx) error {
				err := tx.Bucket([]byte(db.ROOT_BUCKET)).Put([]byte(name), depBytes)
				if err != nil {
					return fmt.Errorf("could not set: %v", err)
				}
				return nil
			})
		}
	}
	return nil
}
