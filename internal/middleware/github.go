package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"context"
	
	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/models"

	"github.com/akyoto/cache"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/google/go-github/v51/github"
	"golang.org/x/oauth2"
)

type OAuthLogin struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
}

var Cache *cache.Cache

func GitHubLoginHandler(c *gin.Context) {
	c.String(http.StatusOK, fmt.Sprintf("Login URL: https://github.com/login/oauth/authorize?scope=read:user&client_id=%s\n", config.Env.GitHubClientId))
	return
}

func GitHubCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	login := OAuthLogin{ClientID: config.Env.GitHubClientId, ClientSecret: config.Env.GitHubClientSecret, Code: code}
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
	ed := gjson.Get(json, "error_description").String()
	eu := gjson.Get(json, "error_uri").String()

	c.HTML(http.StatusOK, "github/callback.tmpl", gin.H{
		"accessToken": at,
		"errorDesc": ed,
		"errorURI": eu,
	})
	return
}

func NewGitHubMiddleware() gin.HandlerFunc {
	Cache = cache.New(5 * time.Minute)
	return func(c *gin.Context) {
		auth := c.Request.Header["Authorization"]
		if len(auth) == 0 {
			c.String(http.StatusBadRequest, fmt.Sprintf("unable to parse bearer token from request"))
			c.Abort()
			return
		}
		authBody := strings.ToLower(auth[0])
		ghAccessToken := strings.Split(authBody, "bearer ")[1]
		if _, found := Cache.Get(ghAccessToken); !found {
			ctx := context.Background()
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: "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"},
			)
			tc := oauth2.NewClient(ctx, ts)

			client := github.NewClient(tc)
			user, _, err := client.Users.Get(ctx, "")
			if err != nil {
				c.String(http.StatusInternalServerError, "failed to get user from github")
				c.Abort()
				return
			} else {
				Cache.Set(ghAccessToken, ghAccessToken, 10 * time.Minute)
			}
			username := *user.Login
			_, err = models.GetUser(username)
			if err != nil {
				newUser := models.User{
					Username: username,
					Role:     config.Env.GitHubUsersDefaultRole,
					Type:     "github",
				}
				err = models.CreateUser(newUser)
				if err != nil {
					c.String(http.StatusInternalServerError, "failed to create a gemfast user from github callback")
					c.Abort()
					return
				}
			}
		} else {
			return
		}
	}
}
