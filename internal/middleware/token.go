package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gemfast/server/internal/db"
	"github.com/gin-gonic/gin"
)

type TokenMiddleware struct {
	acl *ACL
	db  *db.DB
}

func NewTokenMiddleware(acl *ACL, db *db.DB) *TokenMiddleware {
	return &TokenMiddleware{
		acl: acl,
		db:  db,
	}
}

func (t *TokenMiddleware) TokenMiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, token, ok := c.Request.BasicAuth()
		if !ok {
			auth := c.Request.Header["Authorization"]
			if len(auth) == 0 {
				c.String(http.StatusUnauthorized, "unable to parse username and token from request")
				c.Abort()
				return
			}
			s := strings.Split(auth[0], ":")
			if len(s) != 2 {
				c.String(http.StatusUnauthorized, "malformed Authorization header")
				c.Abort()
				return
			}
			username = s[0]
			token = s[1]
		}
		user, err := t.db.GetUser(username)
		if err != nil {
			c.String(http.StatusForbidden, fmt.Sprintf("no user found with username %s", username))
			c.Abort()
			return
		}
		ok = (user.Token == token)
		if ok {
			ok, err = t.acl.Enforce(user.Role, c.Request.URL.Path, c.Request.Method)
			if err != nil {
				c.String(http.StatusInternalServerError, "failed to check access control list")
				c.Abort()
				return
			}
			if ok {
				c.Next()
			} else {
				c.String(http.StatusForbidden, fmt.Sprintf("user does not have access to the request %s %s", c.Request.Method, c.Request.URL.Path))
				c.Abort()
				return
			}
		} else {
			c.String(http.StatusForbidden, "invalid username or token")
			c.Abort()
			return
		}
	}
}

func (t *TokenMiddleware) CreateUserTokenHandler(c *gin.Context) {
	user, ok := c.Get(IdentityKey)
	if !ok {
		c.String(http.StatusInternalServerError, "Failed to get user from context")
		return
	}
	u, ok := user.(*db.User)
	if !ok {
		c.String(http.StatusInternalServerError, "Failed to cast user to db.User")
		return
	}
	token, err := t.db.CreateUserToken(u.Username)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate token for user")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": u.Username,
	})
}
