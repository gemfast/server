package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	jmw "github.com/appleboy/gin-jwt/v2"
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/models"
	"github.com/golang-jwt/jwt"
	"github.com/rs/zerolog/log"

	"github.com/akyoto/cache"
	"github.com/gin-gonic/gin"
	"github.com/juliangruber/go-intersect"
	"github.com/tidwall/gjson"
)

type OAuthLogin struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
}

var Cache *cache.Cache

const GitHubTokenKey = "github_token"

func init() {
	Cache = cache.New(5 * time.Minute)
}

func NewGHHubMiddleware() (*jmw.GinJWTMiddleware, error) {
	authMiddleware, err := jmw.New(&jmw.GinJWTMiddleware{
		Realm:       "zone",
		Key:         []byte(config.Cfg.Auth.LocalAuthSecretKey),
		Timeout:     time.Hour * 12,
		MaxRefresh:  time.Hour * 24,
		IdentityKey: IdentityKey,
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jmw.ExtractClaims(c)
			return &models.User{
				Username:    claims[IdentityKey].(string),
				Role:        claims[RoleKey].(string),
				GitHubToken: claims[GitHubTokenKey].(string),
			}
		},
		TimeFunc: time.Now,
	})

	if err != nil {
		log.Error().Err(err).Msg("JWT error")
	}

	err = authMiddleware.MiddlewareInit()
	return authMiddleware, err
}

func payload(user *models.User) jwt.MapClaims {
	fmt.Println(user)
	return jwt.MapClaims{
		IdentityKey:    user.Username,
		RoleKey:        user.Role,
		GitHubTokenKey: user.GitHubToken,
	}
}

func generateJWTToken(user *models.User) (string, time.Time, error) {
	mw, err := NewGHHubMiddleware()
	if err != nil {
		panic(err)
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

func GitHubLoginHandler(c *gin.Context) {
	c.String(http.StatusOK, fmt.Sprintf("Login URL: https://github.com/login/oauth/authorize?scope=read:user,read:org&client_id=%s\n", config.Cfg.Auth.GitHubClientId))
}

func GitHubCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	login := OAuthLogin{ClientID: config.Cfg.Auth.GitHubClientId, ClientSecret: config.Cfg.Auth.GitHubClientSecret, Code: code}
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
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read POST login/oauth/access_token response from github")
		c.String(http.StatusForbidden, "failed to read response of access token request")
		c.Abort()
		return
	}
	json := string(body)
	at := gjson.Get(json, "access_token").String()
	user, err := authenticateGitHubUser(at)
	if err != nil {
		log.Error().Err(err).Msg("failed to authenticate github user")
		c.String(http.StatusForbidden, fmt.Sprintf("failed to fetch github user with provided token: %v", err))
		c.Abort()
		return
	}
	ed := gjson.Get(json, "error_description").String()
	eu := gjson.Get(json, "error_uri").String()
	jwt, _, err := generateJWTToken(user)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate JWT token")
		c.String(http.StatusInternalServerError, "failed to generate JWT token")
		c.Abort()
		return
	}
	c.HTML(http.StatusOK, "github/callback.tmpl", gin.H{
		"accessToken": jwt,
		"errorDesc":   ed,
		"errorURI":    eu,
	})
}

func authenticateGitHubUser(at string) (*models.User, error) {
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
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GET user response from github: %w", err)
	}
	json := string(body)
	username := gjson.Get(json, "login").String()
	if username == "" {
		return nil, fmt.Errorf("user login not returned from github")
	}
	err = userMemberOfRequiredOrg(at)
	if err != nil {
		return nil, err
	}
	user, err := models.GetUser(username)
	if err != nil {
		newUser := models.User{
			Username:    username,
			Role:        config.Cfg.Auth.DefaultUserRole,
			Type:        "github",
			GitHubToken: at,
		}
		err = models.CreateUser(newUser)
		if err != nil {
			return nil, err
		}
		return &newUser, nil
	} else if user.GitHubToken != at {
		user.GitHubToken = at
		err = models.UpdateUser(user)
		if err != nil {
			return nil, err
		}
	}
	Cache.Set(at, at, 5*time.Minute)
	return &user, nil
}

func userMemberOfRequiredOrg(at string) error {
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
	body, err := ioutil.ReadAll(res.Body)
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
	for _, gho := range config.Cfg.Auth.GitHubUserOrgs {
		requiredOrgs = append(requiredOrgs, strings.ToLower(gho))
	}
	if len(intersect.Hash(requiredOrgs, userOrgs)) != 0 {
		return nil
	}
	return fmt.Errorf("user is not a member of any required github org")
}

func GitHubMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.Request.Header["Authorization"]
		if len(auth) == 0 {
			c.String(http.StatusBadRequest, "missing authorization header")
			c.Abort()
			return
		}
		authFields := strings.Fields(auth[0])
		if len(authFields) != 2 || strings.ToLower(authFields[0]) != "bearer" {
			c.String(http.StatusBadRequest, "unable to parse bearer token from request")
			c.Abort()
			return
		}
		ghAccessToken := authFields[1]
		if _, found := Cache.Get(ghAccessToken); !found {
			_, err := authenticateGitHubUser(ghAccessToken)
			if err != nil {
				c.String(http.StatusForbidden, fmt.Sprintf("failed to fetch github user with provided token: %v", err))
				c.Abort()
				return
			}
		}
	}
}
