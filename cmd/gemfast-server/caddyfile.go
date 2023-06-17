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

var caddyfileCmd = &cobra.Command{
	Use:   "caddyfile",
	Short: "Write the Caddy configuration file to stdout",
	Long:  "Reads the gemfast.hcl config file and outputs the Caddyfile to stdout or to an output file if specified.",
	Run: func(cmd *cobra.Command, args []string) {
		caddyfile()
	},
}

var Output string

func init() {
	rootCmd.AddCommand(caddyfileCmd)
	caddyfileCmd.Flags().StringVarP(&Output, "output", "o", "", "Location to write the Caddyfile to")
}

func caddyfile() {
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
