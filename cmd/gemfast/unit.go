package cmd

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/gscho/gemfast/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const GemfastSystemdTemplate = `
[Unit]
Description=Gemfast Private Rubygems Server
Documentation=https://gemfast.io
After=network.target network-online.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
`

var unitCmd = &cobra.Command{
	Use:   "unit",
	Short: "Generate a systemd unit for the gemfast server",
	Run: func(cmd *cobra.Command, args []string) {
		generate()
	},
}

func init() {
	rootCmd.AddCommand(unitCmd)
}

func generate() {
	m := make(map[string]string)
	m["BinaryPath"] = config.Env.BinPath
	t, err := template.New("systemd_agent").Parse(GemfastSystemdTemplate)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse template")
		os.Exit(1)
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, m)
	if err != nil {
		log.Error().Err(err).Msg("unable to render systemd unit")
		os.Exit(1)
	}
	fmt.Println(tpl.String())
}
