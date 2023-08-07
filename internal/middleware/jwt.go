package middleware

import (
	"time"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"

	jmw "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

type JWTMiddleware struct {
	cfg *config.Config
	acl *ACL
	db  *db.DB
}

const IdentityKey = "id"
const RoleKey = "role"

func NewJWTMiddleware(cfg *config.Config, acl *ACL, db *db.DB) *JWTMiddleware {
	return &JWTMiddleware{
		cfg: cfg,
		acl: acl,
		db:  db,
	}
}

func (m *JWTMiddleware) InitJwtMiddleware() (*jmw.GinJWTMiddleware, error) {
	authMiddleware, err := jmw.New(&jmw.GinJWTMiddleware{
		Realm:       "zone",
		Key:         []byte(m.cfg.Auth.JWTSecretKey),
		Timeout:     time.Hour * 12,
		MaxRefresh:  time.Hour * 24,
		IdentityKey: IdentityKey,
		PayloadFunc: func(data interface{}) jmw.MapClaims {
			if v, ok := data.(*db.User); ok {
				return jmw.MapClaims{
					IdentityKey: v.Username,
					RoleKey:     v.Role,
				}
			} else {
				log.Error().Msg("failed to map jwt claims")
			}
			return jmw.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jmw.ExtractClaims(c)
			return &db.User{
				Username: claims[IdentityKey].(string),
				Role:     claims[RoleKey].(string),
			}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals login
			if err := c.ShouldBind(&loginVals); err != nil {
				return nil, jmw.ErrMissingLoginValues
			}
			user := &db.User{
				Username: loginVals.Username,
				Password: []byte(loginVals.Password),
			}
			authenticated, err := m.db.AuthenticateLocalUser(user)
			if err != nil {
				return nil, jmw.ErrFailedAuthentication
			}
			return authenticated, nil
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			claims := jmw.ExtractClaims(c)
			role := claims[RoleKey].(string)
			ok, err := m.acl.Enforce(role, c.Request.URL.Path, c.Request.Method)
			if err != nil {
				log.Error().Err(err).Msg("failed to check acl")
				return false
			}
			return ok
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
		log.Error().Err(err).Msg("JWT error")
	}

	err = authMiddleware.MiddlewareInit()
	return authMiddleware, err
}
