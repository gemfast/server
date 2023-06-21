package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gemfast/server/internal/models"
	"github.com/gin-gonic/gin"
)

func NewTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, token, ok := c.Request.BasicAuth()
		if !ok {
			auth := c.Request.Header["Authorization"]
			if len(auth) == 0 {
				c.String(http.StatusBadRequest, "unable to parse username and token from request")
				c.Abort()
				return
			}
			s := strings.Split(auth[0], ":")
			if len(s) != 2 {
				c.String(http.StatusBadRequest, "malformed Authorization header")
				c.Abort()
				return
			}
			username = s[0]
			token = s[1]
		}
		user, err := models.GetUser(username)
		if err != nil {
			c.String(http.StatusNotFound, fmt.Sprintf("no user found with username %s", username))
			c.Abort()
			return
		}
		ok = (user.Token == token)
		if ok {
			ok, err = ACL.Enforce(user.Role, c.Request.URL.Path, c.Request.Method)
			if err != nil {
				c.String(http.StatusInternalServerError, "failed to check access control list")
				c.Abort()
				return
			}
			if ok {
				c.Next()
			} else {
				c.String(http.StatusUnauthorized, fmt.Sprintf("user does not have access to the request %s %s", c.Request.Method, c.Request.URL.Path))
				c.Abort()
				return
			}
		} else {
			c.String(http.StatusUnauthorized, "invalid username or token")
			c.Abort()
			return
		}
	}
}

func CreateTokenHandler(c *gin.Context) {
	user, ok := c.Get(IdentityKey)
	if !ok {
		c.String(http.StatusInternalServerError, "Failed to get user from context")
		return
	}
	u, ok := user.(*models.User)
	if !ok {
		c.String(http.StatusInternalServerError, "Failed to cast user to models.User")
		return
	}
	token, err := models.CreateUserToken(u.Username)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate token for user")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": u.Username,
	})
}
