package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"syscall"

	"github.com/gscho/gemfast/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/yaml.v3"
)

var loginCmd = &cobra.Command{
	Use:   "login [username]",
	Short: "Authenticate with the gemfast server",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		login(args[0])
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func writeGemCredentials(dir string, body []byte) {
	fname := fmt.Sprintf("%s/credentials", dir)
	if _, err := os.Stat(fname); errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			panic(err)
		}
		f, err := os.Create(fname)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		j := map[string]string{}
		json.Unmarshal(body, &j)
		delete(j, "code")
		delete(j, "expire")
		j[":gemfast"] = fmt.Sprintf("Bearer %s", j["token"])
		delete(j, "token")
		data, err := yaml.Marshal(&j)
		_, err = f.WriteString(string(data))
		if err != nil {
			panic(err)
		}
	} else {
		data := make(map[interface{}]interface{})

		yfile, err := ioutil.ReadFile(fname)
		if err != nil {
			panic(err)
		}
		err = yaml.Unmarshal(yfile, &data)
		if err != nil {
			panic(err)
		}
		j := map[string]string{}
		json.Unmarshal(body, &j)
		data[":gemfast"] = fmt.Sprintf("Bearer %s", j["token"])
		out, err := yaml.Marshal(&data)
		err = ioutil.WriteFile(fname, out, 0)
		if err != nil {
			panic(err)
		}
	}
}

func login(username string) {
	fmt.Print("password: ")
	pass, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println("'" + string(pass) + "'")
	if err != nil {
		panic(err)
	}
	httpposturl := fmt.Sprintf("%s:%s/login", config.Env.URL, config.Env.Port)
	login := Login{Username: username, Password: string(pass)}
	jsonData, _ := json.Marshal(login)
	request, err := http.NewRequest("POST", httpposturl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		fmt.Println("\nlogin failed")
		os.Exit(1)
	}
	fmt.Println("\nlogin succeeded")

	body, _ := ioutil.ReadAll(response.Body)
	usr, _ := user.Current()
	gemfdir := fmt.Sprintf("%s/.gemfast", usr.HomeDir)
	err = os.MkdirAll(gemfdir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	path := fmt.Sprintf("%s/config.json", gemfdir)
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.WriteString(string(body))
	if err != nil {
		panic(err)
	}
	writeGemCredentials(fmt.Sprintf("%s/.gem", usr.HomeDir), body)
}
