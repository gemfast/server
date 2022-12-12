package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"syscall"
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

func login(username string) {
	fmt.Print("password: ")
	pass, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(err)
	}
	httpposturl := "http://localhost:8080/login"
	login := Login{Username: username, Password: string(pass)}
	jsonData, _ := json.Marshal(login)
	request, error := http.NewRequest("POST", httpposturl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		panic(error)
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	usr, _ := user.Current()
	path := fmt.Sprintf("%s/.gemfast/gemfastconfig.json", usr.HomeDir)
	f, err := os.Create(path)
	defer f.Close()
	f.WriteString(string(body))
}
