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
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("unable to parse username and token"))
				return
			}
			s := strings.Split(auth[0], ":")
			if len(s) != 2 {
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("malformed Authorization header"))
				return	
			}
			username = s[0]
			token = s[1]
		}
		ok = validateToken(username, token)
		if ok {
			c.Next()
		} else {
			c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("invalid username and token"))
			return
		}
	}
}

func validateToken(username string, token string) bool {
	user, err := models.GetUser(username)
	if err != nil {
		return false
	}
	return user.Token == token
}

func CreateTokenHandler(c *gin.Context) {
	user, _ := c.Get(IdentityKey)
	u, _ := user.(*models.User)
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
