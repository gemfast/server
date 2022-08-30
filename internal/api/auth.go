package api

import (
	"time"

	"github.com/gscho/gemfast/internal/models"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

var identityKey = "id"

func initAuthMiddleware() (*jwt.GinJWTMiddleware, error) {
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "gemfast",
		Key:         []byte("secret key"),
		Timeout:     time.Hour,
		MaxRefresh:  time.Hour,
		IdentityKey: identityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*models.User); ok {
				return jwt.MapClaims{
					identityKey: v.Username,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &models.User{
				Username: claims[identityKey].(string),
			}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals login
			if err := c.ShouldBind(&loginVals); err != nil {
				return nil, jwt.ErrMissingLoginValues
			}
			user := models.User{
				Username: loginVals.Username,
				Password: []byte(loginVals.Password),
			}
			authenticated, err := models.AuthenticateLocalUser(user)
			if err != nil {
				return nil, jwt.ErrFailedAuthentication
			}
			if authenticated {
				return user, nil
			}
			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			return true
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
		TokenLookup:   "header: Authorization",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})

	if err != nil {
		log.Error().Err(err).Msg("JWT Error")
	}

	err = authMiddleware.MiddlewareInit()
	return authMiddleware, err
}
