package main

import (
	// "fmt"

	"github.com/gscho/gemfast/internal/indexer"
)

type Gem struct {
	Version string
}

func main() {
	i := indexer.New("/var/gemfast")
	// i.GenerateIndex()
	i.UpdateIndex()
	// marshal.Marshal()
}
