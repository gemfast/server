package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/license"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

const (
	GEM_BUCKET     = "gems"
	KEY_BUCKET     = "keys"
	LICENSE_BUCKET = "license"
	USER_BUCKET    = "users"
)

var BoltDB *bolt.DB

func Connect(l *license.License) error {
	err := os.MkdirAll(config.Cfg.DBDir, os.ModePerm)
	if err != nil {
		log.Logger.Error().Err(err).Msg(fmt.Sprintf("could make db directory %s", config.Cfg.DBDir))
		return err
	}
	dbFile := filepath.Join(config.Cfg.DBDir, "gemfast.db")
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Logger.Error().Err(err).Msg(fmt.Sprintf("could not open %s", dbFile))
		return err
	}
	BoltDB = db
	createBucket(GEM_BUCKET)
	createBucket(KEY_BUCKET)
	createBucket(LICENSE_BUCKET)
	createBucket(USER_BUCKET)
	log.Info().Str("detail", dbFile).Msg("successfully connected to database")
	if l != nil {
		err = persistLicense(l)
		if err != nil {
			log.Error().Err(err).Msg("could not persist license")
			return err
		}
	}
	return nil
}

func createBucket(bucket string) {
	err := BoltDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			log.Error().Err(err).Msg(fmt.Sprintf("could not create %s bucket", bucket))
			return err
		}
		log.Logger.Trace().Msg(fmt.Sprintf("created %s bucket", bucket))
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg(fmt.Sprintf("could not create %s bucket", bucket))
		os.Exit(1)
	}
}

func persistLicense(l *license.License) error {
	err := BoltDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(LICENSE_BUCKET))
		licenseBytes, err := json.Marshal(l)
		if err != nil {
			return fmt.Errorf("could not marshal gem to json: %v", err)
		}
		err = b.Put([]byte(l.Fingerprint), []byte(licenseBytes))
		if err != nil {
			log.Error().Err(err).Msg("could not persist license")
			return err
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("could not persist license")
		return err
	}
	return nil
}
