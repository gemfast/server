package models

import (
	"encoding/json"
	"fmt"

	"github.com/gscho/gemfast/internal/db"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-password/password"
	"github.com/spf13/viper"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string
	Password []byte
}

func userFromBytes(data []byte) (*User, error) {
	var p *User
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func AuthenticateLocalUser(incoming User) (bool, error) {
	current, err := GetUser(incoming.Username)
	if err != nil {
		return false, err
	}
	if err := bcrypt.CompareHashAndPassword(current.Password, incoming.Password); err != nil {
		return false, err
	}
	return true, nil
}

func GetUser(username string) (User, error) {
	var existing []byte
	db.BoltDB.View(func(tx *bolt.Tx) error {
		userBytes := tx.Bucket([]byte(db.USER_BUCKET)).Get([]byte(username))
		existing = userBytes
		return nil
	})
	if len(existing) == 0 {
		return User{}, nil
	}
	user, err := userFromBytes(existing)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal user from bytes")
		return User{}, err
	}
	return *user, nil
}

func CreateAdminIfNotExists() error {
	user, err := GetUser("admin")
	if err != nil {
		panic(err)
	}
	if user.Username != "" && len(user.Password) > 0 {
		if viper.Get("admin_password") == nil {
			return nil
		}
		pw := viper.Get("admin_password").(string)
		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(pw)); err != nil {
			log.Info().Msg("updating admin user password to $GEMFAST_ADMIN_PASSWORD")
			userBytes, err := json.Marshal(User{
				Username: "admin",
				Password: []byte(pw),
			})
			if err != nil {
				return fmt.Errorf("could not marshal user to json: %v", err)
			}
			err = db.BoltDB.Update(func(tx *bolt.Tx) error {
				err = tx.Bucket([]byte(db.USER_BUCKET)).Put([]byte("admin"), userBytes)
				if err != nil {
					return fmt.Errorf("could not set: %v", err)
				}
				return nil
			})
		}
		return nil
	}
	user = User{
		Username: "admin",
		Password: getAdminPassword(),
	}
	userBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("could not marshal user to json: %v", err)
	}
	err = db.BoltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(db.USER_BUCKET)).Put([]byte(user.Username), userBytes)
		if err != nil {
			return fmt.Errorf("could not set: %v", err)
		}
		return nil
	})
	return nil
}

func getAdminPassword() []byte {
	var pw string
	var err error
	if viper.Get("admin_password") == nil {
		pw, err = generatePassword()
		if err != nil {
			panic(err)
		}
	} else {
		pw = viper.Get("admin_password").(string)
	}
	pwbytes, err := bcrypt.GenerateFromPassword([]byte(pw), 14)
	if err != nil {
		panic(err)
	}
	return pwbytes
}

func generatePassword() (string, error) {
	res, err := password.Generate(32, 10, 10, false, false)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate an admin password")
		return "", err
	}
	log.Warn().Msg("generating admin password because environment variable GEMFAST_ADMIN_PASSWORD not set")
	log.Info().Str("password", string(res)).Msg("generated admin password")
	return res, nil
}
