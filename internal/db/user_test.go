package db

import (
	"github.com/gemfast/server/internal/config"
)

func (suite *ModelsTestSuite) TestGetUser() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	user, err := db.GetUser("bobvance")
	suite.Nil(err)
	suite.NotNil(user)
	suite.Equal("bobvance", user.Username)
	suite.Equal("admin", user.Role)
}

func (suite *ModelsTestSuite) TestGetUserNotFound() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	user, err := db.GetUser("notfound")
	suite.Nil(user)
	suite.NotNil(err)
}

func (suite *ModelsTestSuite) TestGetAllUsers() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	users, err := db.GetUsers()
	suite.Nil(err)
	suite.Equal(2, len(users))
	suite.Equal("bobvance", users[0].Username)
	suite.Equal("phyllisvance", users[1].Username)
}

func (suite *ModelsTestSuite) TestAuthenticateLocalUser() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	user, err := db.AuthenticateLocalUser(&User{Username: "bobvance", Password: []byte("mypassword")})
	suite.Nil(err)
	suite.NotNil(user)
	suite.Equal("bobvance", user.Username)
	user, err = db.AuthenticateLocalUser(&User{Username: "bobvance", Password: []byte("notmypassword")})
	suite.Nil(user)
	suite.NotNil(err)
	user, err = db.AuthenticateLocalUser(&User{Username: "notauser", Password: []byte("mypassword")})
	suite.Nil(user)
	suite.NotNil(err)
}

func (suite *ModelsTestSuite) TestCreateUser() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	err := db.CreateUser(&User{Username: "newuser", Password: []byte("newpassword"), Role: "read"})
	suite.Nil(err)
	user, err := db.GetUser("newuser")
	suite.Nil(err)
	suite.NotNil(user)
	suite.Equal("read", user.Role)
}

func (suite *ModelsTestSuite) TestCreateAdminUserIfNotExists() {
	cfg := config.NewConfig()
	db := NewTestDB(suite.db, cfg)
	err := db.CreateAdminUserIfNotExists()
	suite.Nil(err)
	user, err := db.GetUser("admin")
	suite.Nil(err)
	suite.NotNil(user)
	suite.Equal("admin", user.Role)
	suite.NotNil(user.Password)
}

func (suite *ModelsTestSuite) TestGetAdminPassword() {
	cfg := config.NewConfig()
	cfg.Auth.AdminPassword = "mypassword"
	db := NewTestDB(suite.db, cfg)
	pw, err := db.getAdminPassword()
	suite.Nil(err)
	suite.NotNil(pw)
	cfg.Auth.AdminPassword = ""
	generated, err := db.getAdminPassword()
	suite.Nil(err)
	suite.NotNil(generated)
	suite.NotEqual(pw, generated)
}
