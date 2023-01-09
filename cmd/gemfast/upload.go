package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"github.com/gscho/gemfast/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload [gem]",
	Short: "Upload a gem to the gemfast server",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		upload(args[0])
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

type GemfastConfig struct {
	Code   string `json:"code"`
	Expire string `json:"expire"`
	Token  string `json:"token"`
}

func upload(gem string) {
	url := fmt.Sprintf("%s:%s", config.Env.URL, config.Env.Port)
	if _, err := os.Stat(gem); errors.Is(err, os.ErrNotExist) {
		log.Error().Str("path", gem).Err(err).Msg("gem not found at given path")
	}
	file, _ := os.Open(gem)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.Close()

	r, _ := http.NewRequest("POST", url+"/upload", body)
	r.Header.Add("Content-Type", writer.FormDataContentType())
	if config.Env.AuthMode == "local" {
		usr, err := user.Current()
		if err != nil {
			log.Error().Err(err).Msg("unable to get current user")
			os.Exit(1)
		}
		gemfdir := fmt.Sprintf("%s/.gemfast/config.json", usr.HomeDir)
		if _, err := os.Stat(gemfdir); errors.Is(err, os.ErrNotExist) {
			log.Error().Str("path", gem).Err(err).Msg("gemfast config file not found. have you logged in first?")
			os.Exit(1)
		}
		fileBytes, err := os.ReadFile(gemfdir)
		if err != nil {
			log.Error().Err(err).Msg("unable to read gemfast config file")
			os.Exit(1)
		}
		var gemfConfig GemfastConfig
		err = json.Unmarshal(fileBytes, &gemfConfig)
		r.Header.Add("Authorization", "Bearer "+gemfConfig.Token)
	}
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		log.Error().Err(err).Msg("error making http request")
		os.Exit(1)
	}
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Msg("could not read response body")
		os.Exit(1)
	}
	if res.StatusCode >= 300 {
		log.Error().Str("response", string(resBody)).Msg("error making http request")
		os.Exit(1)
	} else {
		log.Info().Str("gem", gem).Msg(string(resBody))
	}
}
