package models

import (
	"github.com/gemfast/server/internal/config"
)

func (suite *ModelsTestSuite) TestGetUser() {
	user, err := GetUser("bobvance")
	suite.Nil(err)
	suite.NotNil(user)
	suite.Equal("bobvance", user.Username)
	suite.Equal("admin", user.Role)
}

func (suite *ModelsTestSuite) TestGetUserNotFound() {
	user, err := GetUser("notfound")
	suite.Nil(user)
	suite.NotNil(err)
}

func (suite *ModelsTestSuite) TestGetAllUsers() {
	users, err := GetUsers()
	suite.Nil(err)
	suite.Equal(2, len(users))
	suite.Equal("bobvance", users[0].Username)
	suite.Equal("phyllisvance", users[1].Username)
}

func (suite *ModelsTestSuite) TestAuthenticateLocalUser() {
	user, err := AuthenticateLocalUser(&User{Username: "bobvance", Password: []byte("mypassword")})
	suite.Nil(err)
	suite.NotNil(user)
	suite.Equal("bobvance", user.Username)
	user, err = AuthenticateLocalUser(&User{Username: "bobvance", Password: []byte("notmypassword")})
	suite.Nil(user)
	suite.NotNil(err)
	user, err = AuthenticateLocalUser(&User{Username: "notauser", Password: []byte("mypassword")})
	suite.Nil(user)
	suite.NotNil(err)
}

func (suite *ModelsTestSuite) TestCreateUser() {
	err := CreateUser(&User{Username: "newuser", Password: []byte("newpassword"), Role: "read"})
	suite.Nil(err)
	user, err := GetUser("newuser")
	suite.Nil(err)
	suite.NotNil(user)
	suite.Equal("read", user.Role)
}

func (suite *ModelsTestSuite) TestCreateAdminUserIfNotExists() {
	config.LoadConfig()
	err := CreateAdminUserIfNotExists()
	suite.Nil(err)
	user, err := GetUser("admin")
	suite.Nil(err)
	suite.NotNil(user)
	suite.Equal("admin", user.Role)
	suite.NotNil(user.Password)
}

func (suite *ModelsTestSuite) TestGetAdminPassword() {
	config.LoadConfig()
	config.Cfg.Auth.AdminPassword = "mypassword"
	pw, err := getAdminPassword()
	suite.Nil(err)
	suite.NotNil(pw)
	config.Cfg.Auth.AdminPassword = ""
	generated, err := getAdminPassword()
	suite.Nil(err)
	suite.NotNil(generated)
	suite.NotEqual(pw, generated)
}
