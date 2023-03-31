package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gemfast/server/internal/config"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
)

type OAuthLogin struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
}

func GitHubLoginHandler(c *gin.Context) {
	c.String(http.StatusOK, fmt.Sprintf("https://github.com/login/oauth/authorize?scope=read:user&client_id=%s", config.Env.GitHubClientId))
	return
}

func GitHubCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	login := OAuthLogin{ClientID: config.Env.GitHubClientId, ClientSecret: config.Env.GitHubClientSecret, Code: code}
	jsonData, _ := json.Marshal(login)
	bodyReader := bytes.NewBuffer(jsonData)
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bodyReader)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to get an access token from github")
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	at := string(body)

	c.HTML(http.StatusOK, "github/callback.tmpl", gin.H{
		"accessToken": at,
	})
	return
}
