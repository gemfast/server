package middleware

import (
	b64 "encoding/base64"
	"fmt"
	"net/http"

	"github.com/gemfast/server/internal/models"
	"github.com/gin-gonic/gin"
)

func GinTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, token, ok := c.Request.BasicAuth()
		if !ok {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("unable to parse username and token"))
			return
		}
		ok = validateToken(username, token)
		if ok {
			c.Next()
		} else {
			c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("unable to parse username and token"))
			return
		}
	}
}

func validateToken(username string, token string) bool {
	user, err := models.GetUser(username)
	if err != nil {
		return false
	}
	decoded, _ := b64.StdEncoding.DecodeString(token)
	return user.Token == string(decoded)
}
