package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GinTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, p, ok := c.Request.BasicAuth()
		if !ok {
			c.String(http.StatusBadRequest, "unable to parse username and token")
      return
		}
		ok = validateToken(u, p)
		if ok {
     c.Next() 
    } else {
    	c.String(http.StatusUnauthorized, "invalid api token provided")
      return
    }
	}
}

func validateToken(user string, pass string) (bool) {
	return pass == "lol"
}
