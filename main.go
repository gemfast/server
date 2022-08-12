package main

import (
	"github.com/gscho/gemfast/internal/api"
)

func main() {
	err := api.Run(); if err != nil {
		panic(err)
	}
}
