package indexer

import (
	"testing"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
)

func TestGemList(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Dir = "/tmp"
	db, _ := db.NewDB(cfg)
	cfg.GemDir = "../../test/fixtures/gem_list"
	i, err := NewIndexer(cfg, db)
	if err != nil {
		t.Errorf("Expected no error. Received %s", err)
	}
	expected := []string{"../../test/fixtures/gem_list/a.gem", "../../test/fixtures/gem_list/another_@gem]2345.gem", "../../test/fixtures/gem_list/b.gem"}
	actual, err := i.gemList()
	if err != nil {
		t.Errorf("Expected no error. Received %s", err)
	}
	if len(actual) != len(expected) {
		t.Errorf("Expected gem list with length %d. Received gem list with length %d", len(expected), len(actual))
	}
	for i, gem := range expected {
		if actual[i] != gem {
			t.Errorf("Expected %s at index %d. Received gem %s", gem, i, actual[i])
		}
	}
}
