package db

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-password/password"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username    string `json:"username"`
	Password    []byte `json:"password,omitempty"`
	Token       string `json:"token,omitempty"`
	Role        string `json:"role"`
	Type        string `json:"type"`
	GitHubToken string `json:"github_token,omitempty"`
}

func ValidUserRoles() []string {
	return []string{"admin", "read", "write"}
}

func userFromBytes(data []byte) (*User, error) {
	var u *User
	err := json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) AuthenticateLocalUser(incoming *User) (*User, error) {
	current, err := db.GetUser(incoming.Username)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword(current.Password, incoming.Password); err != nil {
		return nil, err
	}
	return current, nil
}

func (db *DB) GetUser(username string) (*User, error) {
	var existing []byte
	db.boltDB.View(func(tx *bolt.Tx) error {
		userBytes := tx.Bucket([]byte(UserBucket)).Get([]byte(username))
		existing = userBytes
		return nil
	})
	if len(existing) == 0 {
		return nil, fmt.Errorf("user %s not found", username)
	}
	user, err := userFromBytes(existing)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal user from bytes")
		return nil, err
	}
	return user, nil
}

func (db *DB) GetUsers() ([]*User, error) {
	var users []*User
	err := db.boltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(UserBucket))
		b.ForEach(func(k, v []byte) error {
			user, err := userFromBytes(v)
			if err != nil {
				return err
			}
			users = append(users, user)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (db *DB) CreateUser(user *User) error {
	userBytes, err := json.Marshal(user)
	err = db.boltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(UserBucket)).Put([]byte(user.Username), userBytes)
		if err != nil {
			return fmt.Errorf("could not set: %v", err)
		}
		return nil
	})
	return err
}

func (db *DB) CreateAdminUserIfNotExists() error {
	user, err := db.GetUser("admin")
	if err != nil {
		log.Trace().Msg("admin user not found")
	}
	if user != nil && user.Username != "" && len(user.Password) > 0 {
		if db.cfg.Auth.AdminPassword == "" {
			return nil
		}
		pw := db.cfg.Auth.AdminPassword
		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(pw)); err != nil {
			log.Info().Msg("updating admin user password")
		} else {
			return nil
		}
	}
	pw, err := db.getAdminPassword()
	if err != nil {
		return err
	}
	user = &User{
		Username: "admin",
		Password: pw,
		Role:     "admin",
		Type:     "local",
	}
	log.Trace().Msg("here")
	userBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("could not marshal user to json: %v", err)
	}
	err = db.boltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(UserBucket)).Put([]byte(user.Username), userBytes)
		if err != nil {
			return fmt.Errorf("could not set: %v", err)
		}
		return nil
	})
	return nil
}

func (db *DB) CreateLocalUsers() error {
	if len(db.cfg.Auth.LocalUsers) == 0 {
		log.Trace().Msg("no local users to add")
		return nil
	}
	var users map[string]*User = make(map[string]*User)
	db.boltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(UserBucket))
		if b == nil {
			return fmt.Errorf("get bucket: FAILED")
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if string(k) != "admin" {
				user, err := userFromBytes(v)
				if err != nil {
					return fmt.Errorf("failed reading existing local users: %v", err)
				}
				users[string(k)] = user
			}
		}
		return nil
	})

	m := make(map[string]bool)
	db.boltDB.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(UserBucket))
		for _, u := range db.cfg.Auth.LocalUsers {
			username := u.Username
			pw := u.Password
			role := u.Role
			if role == "" {
				role = db.cfg.Auth.DefaultUserRole
			}
			pwbytes, err := bcrypt.GenerateFromPassword([]byte(pw), db.cfg.Auth.BcryptCost)
			if err != nil {
				panic(err)
			}

			userData, exists := users[username]
			if exists {
				userData.Password = pwbytes
				userData.Role = role
				userData.Type = "local"
			} else {
				userData = &User{
					Username: username,
					Password: pwbytes,
					Role:     role,
					Type:     "local",
				}
			}
			m[username] = true
			userBytes, err := json.Marshal(userData)
			if err != nil {
				return fmt.Errorf("could not marshal user to json: %v", err)
			}
			err = b.Put([]byte(username), userBytes)
			if err != nil {
				return fmt.Errorf("could not set: %v", err)
			}
			log.Trace().Str("detail", username).Msg("added or modified user")
		}
		b = tx.Bucket([]byte(UserBucket))
		for username := range users {
			if !m[username] {
				log.Trace().Str("detail", username).Msg("removed user")
				b.Delete([]byte(username))
			}
		}
		return nil
	})
	return nil
}

func (db *DB) getAdminPassword() ([]byte, error) {
	var pw string
	var err error
	if db.cfg.Auth.AdminPassword == "" {
		pw, err = generatePassword()
		if err != nil {
			return nil, err
		}
	} else {
		pw = db.cfg.Auth.AdminPassword
	}
	pwbytes, err := bcrypt.GenerateFromPassword([]byte(pw), db.cfg.Auth.BcryptCost)
	if err != nil {
		return nil, err
	}
	return pwbytes, nil
}

func generatePassword() (string, error) {
	pw, err := password.Generate(32, 10, 0, false, false)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate an admin password")
		return "", err
	}
	log.Warn().Msg("generating admin password because admin_password not set")
	log.Info().Str("detail", pw).Msg("generated admin password")
	return pw, nil
}

func (db *DB) CreateUserToken(username string) (string, error) {
	user, err := db.GetUser(username)
	if err != nil {
		log.Error().Err(err).Msg("failed to get existing user")
		return "", err
	}
	token, err := password.Generate(32, 10, 0, true, false)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate a token")
		return "", err
	}
	user.Token = token
	userBytes, err := json.Marshal(user)
	if err != nil {
		return "", fmt.Errorf("could not marshal user to json: %v", err)
	}
	err = db.boltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(UserBucket)).Put([]byte(user.Username), userBytes)
		if err != nil {
			return fmt.Errorf("could not set: %v", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func (db *DB) UpdateUser(user *User) error {
	ok := func(user *User) bool {
		for _, role := range ValidUserRoles() {
			if role == user.Role {
				return true
			}
		}
		return false
	}(user)
	if !ok {
		return fmt.Errorf("role %s is not a valid role", user.Role)
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("could not marshal user to json: %v", err)
	}
	err = db.boltDB.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket([]byte(UserBucket)).Put([]byte(user.Username), userBytes)
		if err != nil {
			return fmt.Errorf("could not set: %v", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) DeleteUser(username string) (bool, error) {
	deleted := false
	err := db.boltDB.Update(func(tx *bolt.Tx) error {
		err := tx.Bucket([]byte(UserBucket)).Delete([]byte(username))
		if err != nil {
			return fmt.Errorf("could not delete: %v", err)
		}
		return nil
	})
	if err != nil {
		return deleted, err
	} else {
		deleted = true
	}
	return deleted, nil
}
