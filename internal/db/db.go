package db

import (
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

const ROOT_BUCKET = "root"

var BoltDB *bolt.DB

func Connect() error {
	db, err := bolt.Open("dev/gemfast.db", 0600, nil)
	if err != nil {
		log.Logger.Error().Err(err).Msg("could not open dev/gemfast.db")
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(ROOT_BUCKET))
		if err != nil {
			log.Error().Err(err).Msg("could not create root bucket")
			return err
		}
		return nil
	})
	BoltDB = db
	return nil
}
