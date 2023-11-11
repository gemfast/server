package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"strings"
	"time"

	jmw "github.com/appleboy/gin-jwt/v2"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gin-contrib/sessions"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"github.com/juliangruber/go-intersect"
	"github.com/tidwall/gjson"
)

type OAuthLogin struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
}

type GitHubMiddleware struct {
	cfg      *config.Config
	acl      *ACL
	tokenKey string
	db       *db.DB
}

const GitHubTokenKey = "github_token"

func NewGitHubMiddleware(cfg *config.Config, acl *ACL, db *db.DB) *GitHubMiddleware {
	return &GitHubMiddleware{
		cfg:      cfg,
		tokenKey: GitHubTokenKey,
		acl:      acl,
		db:       db,
	}
}

func (ghm *GitHubMiddleware) InitGitHubMiddleware() (*jmw.GinJWTMiddleware, error) {
	authMiddleware, err := jmw.New(&jmw.GinJWTMiddleware{
		Realm:       "zone",
		Key:         []byte(ghm.cfg.Auth.JWTSecretKey),
		Timeout:     time.Hour * 12,
		MaxRefresh:  time.Hour * 24,
		IdentityKey: IdentityKey,
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jmw.ExtractClaims(c)
			return &db.User{
				Username:    claims[IdentityKey].(string),
				Role:        claims[RoleKey].(string),
				GitHubToken: claims[GitHubTokenKey].(string),
			}
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

func payload(user *db.User) jwt.MapClaims {
	return jwt.MapClaims{
		IdentityKey:    user.Username,
		RoleKey:        user.Role,
		GitHubTokenKey: user.GitHubToken,
	}
}

func (ghm *GitHubMiddleware) generateJWTToken(user *db.User) (string, time.Time, error) {
	mw, err := ghm.InitGitHubMiddleware()
	if err != nil {
		return "", time.Time{}, err
	}
	token := jwt.New(jwt.GetSigningMethod(mw.SigningAlgorithm))
	claims := token.Claims.(jwt.MapClaims)

	for key, value := range payload(user) {
		claims[key] = value
	}

	expire := mw.TimeFunc().Add(mw.Timeout)
	claims["exp"] = expire.Unix()
	claims["orig_iat"] = mw.TimeFunc().Unix()
	tokenString, err := token.SignedString(mw.Key)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expire, nil
}

func (ghm *GitHubMiddleware) GitHubLoginHandler(c *gin.Context) {
	c.String(http.StatusOK, fmt.Sprintf("Login URL: https://github.com/login/oauth/authorize?scope=read:user,read:org&client_id=%s\n", ghm.cfg.Auth.GitHubClientId))
}

func (ghm *GitHubMiddleware) GitHubCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	login := OAuthLogin{ClientID: ghm.cfg.Auth.GitHubClientId, ClientSecret: ghm.cfg.Auth.GitHubClientSecret, Code: code}
	jsonData, _ := json.Marshal(login)
	bodyReader := bytes.NewBuffer(jsonData)
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bodyReader)
	if err != nil {
		log.Error().Err(err).Msg("failed to create POST login/oauth/access_token request")
		c.String(http.StatusInternalServerError, "failed to create POST login/oauth/access_token request")
		c.Abort()
		return
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get an access token from github")
		c.String(http.StatusInternalServerError, "failed to get an access token from github")
		c.Abort()
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read POST login/oauth/access_token response from github")
		c.String(http.StatusForbidden, "failed to read response of access token request")
		c.Abort()
		return
	}
	json := string(body)
	at := gjson.Get(json, "access_token").String()
	user, err := ghm.authenticateGitHubUser(at)
	if err != nil {
		log.Error().Err(err).Msg("failed to authenticate github user")
		c.String(http.StatusForbidden, fmt.Sprintf("failed to fetch github user with provided token: %v", err))
		c.Abort()
		return
	}
	// ed := gjson.Get(json, "error_description").String()
	// eu := gjson.Get(json, "error_uri").String()
	jwt, _, err := ghm.generateJWTToken(user)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate JWT token")
		c.String(http.StatusInternalServerError, "failed to generate JWT token")
		c.Abort()
		return
	}
	session := sessions.Default(c)
	session.Set("authToken", jwt)
	session.Save()
	// c.HTML(http.StatusOK, "github/callback", gin.H{
	// 	"accessToken": jwt,
	// 	"errorDesc":   ed,
	// 	"errorURI":    eu,
	// })
	c.Redirect(http.StatusFound, "/ui")
	c.Abort()
}

func (ghm *GitHubMiddleware) authenticateGitHubUser(at string) (*db.User, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+at)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	} else if res.StatusCode >= 400 {
		return nil, fmt.Errorf("github returned an error")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GET user response from github: %w", err)
	}
	json := string(body)
	username := gjson.Get(json, "login").String()
	if username == "" {
		return nil, fmt.Errorf("user login not returned from github")
	}
	err = ghm.userMemberOfRequiredOrg(at)
	if err != nil {
		return nil, err
	}
	user, err := ghm.db.GetUser(username)
	if err != nil {
		newUser := &db.User{
			Username:    username,
			Role:        ghm.cfg.Auth.DefaultUserRole,
			Type:        "github",
			GitHubToken: at,
		}
		err = ghm.db.CreateUser(newUser)
		if err != nil {
			return nil, err
		}
		return newUser, nil
	} else if user.GitHubToken != at {
		user.GitHubToken = at
		err = ghm.db.UpdateUser(user)
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (ghm *GitHubMiddleware) userMemberOfRequiredOrg(at string) error {
	req, err := http.NewRequest("GET", "https://api.github.com/user/orgs", nil)
	if err != nil {
		return fmt.Errorf("failed to create GET user/orgs request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+at)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	} else if res.StatusCode >= 400 {
		return fmt.Errorf("github returned an error")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read GET user/orgs response from github: %w", err)
	}
	json := string(body)
	result := gjson.Get(json, "#.login")
	var userOrgs []string
	for _, name := range result.Array() {
		userOrgs = append(userOrgs, strings.ToLower(name.String()))
	}

	var requiredOrgs []string
	for _, gho := range ghm.cfg.Auth.GitHubUserOrgs {
		requiredOrgs = append(requiredOrgs, strings.ToLower(gho))
	}
	if len(intersect.Hash(requiredOrgs, userOrgs)) != 0 {
		return nil
	}
	return fmt.Errorf("user is not a member of any required github org")
}

func (ghm *GitHubMiddleware) GitHubMiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		var jwtToken *jwt.Token
		var err error
		var browser bool
		userAgent := c.GetHeader("User-Agent")
		browser = strings.HasPrefix(userAgent, "Mozilla")
		if browser {
			jwtToken, err = ghm.getSessionAuth(c)
			if err != nil {
				log.Error().Err(err).Msg("failed to get jwt token from session")
				c.HTML(http.StatusOK, "github/login", gin.H{
					"clientID": ghm.cfg.Auth.GitHubClientId,
				})
				c.Abort()
				return
			}
		} else {
			jwtToken, err = ghm.getHeaderAuth(c)
			if err != nil {
				log.Error().Err(err).Msg("failed to get jwt token from request")
				ghm.GitHubLoginHandler(c)
				c.Abort()
				return
			}
		}

		claims := jmw.ExtractClaimsFromToken(jwtToken)
		if len(claims) == 0 {
			c.String(http.StatusForbidden, "unable to extract claims from jwt token")
			c.Abort()
			return
		}
		role := claims[RoleKey].(string)
		ghAccessToken := claims[GitHubTokenKey].(string)
		_, err = ghm.authenticateGitHubUser(ghAccessToken)
		if err != nil {
			log.Error().Err(err).Msg("failed to authenticate github user")
			if browser {
				c.HTML(http.StatusOK, "github/login", gin.H{})
				c.Abort()
				return
			}
			ghm.GitHubLoginHandler(c)
			c.Abort()
			return
		}
		ok, err := ghm.acl.Enforce(role, c.Request.URL.Path, c.Request.Method)
		if err != nil {
			log.Error().Err(err).Msg("failed to check acl")
			c.String(http.StatusForbidden, "failed to check acl")
			c.Abort()
			return
		}
		if !ok {
			c.String(http.StatusForbidden, "denied access to resource by ACL")
			c.Abort()
			return
		}
	}
}

func (ghm *GitHubMiddleware) GitHubLogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/ui")
	c.Abort()
}

func (ghm *GitHubMiddleware) getHeaderAuth(c *gin.Context) (*jwt.Token, error) {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("no authorization header found")
	}
	authFields := strings.Split(auth, " ")
	if len(authFields) != 2 || strings.ToLower(authFields[0]) != "bearer" {
		return nil, fmt.Errorf("invalid authorization header")
	}
	return ghm.stringToJWT(authFields[1])
}

func (ghm *GitHubMiddleware) getSessionAuth(c *gin.Context) (*jwt.Token, error) {
	session := sessions.Default(c)
	sessionAuth := session.Get("authToken")
	switch tokenString := sessionAuth.(type) {
	case string:
		return ghm.stringToJWT(tokenString)
	default:
		return nil, fmt.Errorf("invalid session auth token")
	}
}

func (ghm *GitHubMiddleware) stringToJWT(tokenString string) (*jwt.Token, error) {
	jwtToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if jwt.GetSigningMethod("HS256") != t.Method {
			return nil, jmw.ErrInvalidSigningAlgorithm
		}
		return []byte(ghm.cfg.Auth.JWTSecretKey), nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to parse jwt token from request")
		return nil, fmt.Errorf("failed to parse jwt token from request")
	}
	if !jwtToken.Valid {
		return nil, fmt.Errorf("invalid jwt token")
	}
	return jwtToken, nil
}
