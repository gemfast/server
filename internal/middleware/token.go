package middleware

import (
	"net/http"
	b64 "encoding/base64"

	"github.com/gin-gonic/gin"
	"github.com/gemfast/server/internal/models"
)

func GinTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, token, ok := c.Request.BasicAuth()
		if !ok {
			c.String(http.StatusBadRequest, "unable to parse username and token")
      return
		}
		ok = validateToken(username, token)
		if ok {
     c.Next() 
    } else {
    	c.String(http.StatusUnauthorized, "invalid api token provided")
      return
    }
	}
}

func validateToken(username string, token string) (bool) {
	user, err := models.GetUser(username) 
	if err != nil {
		return false
	}
	decoded, _ := b64.StdEncoding.DecodeString(token)
	return user.Token == string(decoded)
}
