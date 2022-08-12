package main

import (
	"github.com/gscho/gemfast/internal/api"
	zl "github.com/rs/zerolog"
)

func init() {
	zl.TimeFieldFormat = zl.TimeFormatUnix
	zl.SetGlobalLevel(zl.DebugLevel)
}

func main() {
	err := api.Run(); if err != nil {
		panic(err)
	}
}
