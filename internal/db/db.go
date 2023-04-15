package db

import (
	"fmt"
	"github.com/gemfast/server/internal/config"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
)

const (
	GEM_DEPENDENCY_BUCKET = "gem_dependencies"
	USER_BUCKET           = "users"
)

var BoltDB *bolt.DB

func Connect() error {
	err := os.MkdirAll(config.Env.DBDir, os.ModePerm)
	if err != nil {
		log.Logger.Error().Err(err).Msg(fmt.Sprintf("could make db directory %s", config.Env.DBDir))
		return err
	}
	dbFile := filepath.Join(config.Env.DBDir, "gemfast.db")
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Logger.Error().Err(err).Msg(fmt.Sprintf("could not open %s", dbFile))
		return err
	}
	BoltDB = db
	createBucket(GEM_DEPENDENCY_BUCKET)
	createBucket(USER_BUCKET)
	log.Info().Str("db", dbFile).Msg("successfully connected to database")
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
		panic(err)
	}
}
