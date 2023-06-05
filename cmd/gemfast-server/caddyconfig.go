package cmd

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/gemfast/server/internal/config"
	"github.com/spf13/cobra"
)

const CaddyfileTemplate = `{{ .URL }}:{{ .CaddyPort }} 

encode zstd gzip
reverse_proxy :{{ .Port }}`

var caddyCfgCmd = &cobra.Command{
	Use:   "caddy-config",
	Short: "Output the Caddy config",
	Long:  "Reads the gemfast.hcl config file and outputs the Caddy config to stdout",
	Run: func(cmd *cobra.Command, args []string) {
		caddyConfig()
	},
}

var Output string

func init() {
	rootCmd.AddCommand(caddyCfgCmd)
	caddyCfgCmd.Flags().StringVarP(&Output, "output", "o", "", "Location to write the Caddyfile to")
}

func caddyConfig() {
	m := make(map[string]interface{})
	m["URL"] = config.Cfg.URL
	m["CaddyPort"] = config.Cfg.CaddyPort
	m["Port"] = config.Cfg.Port
	t, err := template.New("Caddyfile").Parse(CaddyfileTemplate)
	check(err)

	var tpl bytes.Buffer
	err = t.Execute(&tpl, m)
	check(err)
	if Output != "" {
		err := os.WriteFile(Output, tpl.Bytes(), 0644)
		check(err)
		return
	}
	fmt.Println(tpl.String())
}
