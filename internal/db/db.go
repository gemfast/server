package db

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/license"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

const (
	GemBucket     = "gems"
	KeyBucket     = "keys"
	LicenseBucket = "license"
	UserBucket    = "users"
)

type DB struct {
	boltDB *bolt.DB
	dbFile string
	cfg    *config.Config
}

func NewTestDB(boltDB *bolt.DB, cfg *config.Config) *DB {
	return &DB{boltDB: boltDB, cfg: cfg}
}

func NewDB(cfg *config.Config) (*DB, error) {
	err := os.MkdirAll(cfg.DBDir, os.ModePerm)
	if err != nil {
		log.Logger.Error().Err(err).Msg(fmt.Sprintf("could make db directory %s", cfg.DBDir))
		return nil, err
	}
	dbFile := filepath.Join(cfg.DBDir, "gemfast.db")
	return &DB{dbFile: dbFile, cfg: cfg}, nil
}

func (db *DB) Open() {
	boltDB, err := bolt.Open(db.dbFile, 0600, nil)
	if err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("could not open %s", db.dbFile))
	}
	log.Info().Str("detail", db.dbFile).Msg("successfully connected to database")
	db.boltDB = boltDB
	db.createBucket(GemBucket)
	db.createBucket(KeyBucket)
	db.createBucket(LicenseBucket)
	db.createBucket(UserBucket)
}

func (db *DB) Close() error {
	return db.boltDB.Close()
}

func (db *DB) Backup(w http.ResponseWriter) error {
	return db.boltDB.View(func(tx *bolt.Tx) error {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="gemfast.db"`)
		w.Header().Set("Content-Length", strconv.Itoa(int(tx.Size())))
		_, err := tx.WriteTo(w)
		return err
	})
}

func (db *DB) createBucket(bucket string) {
	err := db.boltDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			log.Error().Err(err).Msg(fmt.Sprintf("could not create %s bucket", bucket))
			return err
		}
		log.Logger.Trace().Msg(fmt.Sprintf("created %s bucket", bucket))
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("could not create %s bucket", bucket))
	}
}

func (db *DB) SaveLicense(l *license.License) error {
	err := db.boltDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(LicenseBucket))
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
