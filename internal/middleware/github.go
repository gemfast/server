package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/models"

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

func init() {
	Cache = cache.New(2 * time.Minute)
}

func GitHubLoginHandler(c *gin.Context) {
	c.String(http.StatusOK, fmt.Sprintf("Login URL: https://github.com/login/oauth/authorize?scope=read:user,read:org&client_id=%s\n", config.Cfg.Auth.GitHubClientId))
	return
}

func GitHubCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	login := OAuthLogin{ClientID: config.Cfg.Auth.GitHubClientId, ClientSecret: config.Cfg.Auth.GitHubClientSecret, Code: code}
	jsonData, _ := json.Marshal(login)
	bodyReader := bytes.NewBuffer(jsonData)
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bodyReader)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to get an access token from github")
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	json := string(body)
	at := gjson.Get(json, "access_token").String()
	err = authenticateGitHubUser(at)
	if err != nil {
		c.String(http.StatusForbidden, fmt.Sprintf("failed to fetch github user with provided token: %v", err))
		c.Abort()
		return
	}
	ed := gjson.Get(json, "error_description").String()
	eu := gjson.Get(json, "error_uri").String()

	c.HTML(http.StatusOK, "github/callback.tmpl", gin.H{
		"accessToken": at,
		"errorDesc":   ed,
		"errorURI":    eu,
	})
	return
}

func authenticateGitHubUser(at string) error {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+at)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	} else if res.StatusCode >= 400 {
		return fmt.Errorf("github returned an error")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	json := string(body)
	username := gjson.Get(json, "login").String()
	if username == "" {
		return fmt.Errorf("user login not returned from github")
	}
	err = userMemberOfRequiredOrg(at)
	if err != nil {
		return err
	}
	_, err = models.GetUser(username)
	if err != nil {
		newUser := models.User{
			Username: username,
			Role:     config.Cfg.Auth.DefaultUserRole,
			Type:     "github",
		}
		err = models.CreateUser(newUser)
		if err != nil {
			return err
		}
	}
	Cache.Set(at, at, 5*time.Minute)
	return nil
}

func userMemberOfRequiredOrg(at string) error {
	req, err := http.NewRequest("GET", "https://api.github.com/user/orgs", nil)
	req.Header.Set("Authorization", "Bearer "+at)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	} else if res.StatusCode >= 400 {
		return fmt.Errorf("github returned an error")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	json := string(body)
	result := gjson.Get(json, "#.login")
	var userOrgs []string
	for _, name := range result.Array() {
		userOrgs = append(userOrgs, strings.ToLower(name.String()))
	}
	ghUserOrgs := strings.ToLower(config.Cfg.Auth.GitHubUserOrgs)
	requiredOrgs := strings.Split(ghUserOrgs, ",")
	if len(intersect.Hash(requiredOrgs, userOrgs)) != 0 {
		return nil
	}
	return fmt.Errorf("user is not a member of any required github org")
}

func NewGitHubMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.Request.Header["Authorization"]
		if len(auth) == 0 {
			c.String(http.StatusBadRequest, fmt.Sprintf("missing authorization header"))
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
			err := authenticateGitHubUser(ghAccessToken)
			if err != nil {
				c.String(http.StatusForbidden, fmt.Sprintf("failed to fetch github user with provided token: %v", err))
				c.Abort()
				return
			}
		}
	}
}
