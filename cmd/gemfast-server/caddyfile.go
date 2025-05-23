package cmd

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/gemfast/server/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const CaddyfileTemplate = `{{- if .AdminDisabled }}{
	admin off
}
{{- end }}
{{ .Host }}:{{ .Port }} {
	encode zstd gzip
	{{- if .MetricsEnabled }}
	metrics /caddymetrics
	{{- end }}
	reverse_proxy :{{ .GemfastPort }}
}
`

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
	cfg := config.NewConfig()
	m := make(map[string]interface{})
	m["Host"] = cfg.CaddyConfig.Host
	m["Port"] = cfg.CaddyConfig.Port
	m["GemfastPort"] = cfg.Port
	if !cfg.CaddyConfig.AdminAPIEnabled {
		m["AdminDisabled"] = true
	}
	if !cfg.CaddyConfig.MetricsDisabled {
		m["MetricsEnabled"] = true
	}
	t, err := template.New("Caddyfile").Parse(CaddyfileTemplate)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse Caddyfile template")
	}

	var tpl bytes.Buffer
	err = t.Execute(&tpl, m)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to execute Caddyfile template")
	}
	if Output != "" {
		err := os.WriteFile(Output, tpl.Bytes(), 0644)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to write Caddyfile")
		}
		return
	}
	fmt.Println(tpl.String())
}
