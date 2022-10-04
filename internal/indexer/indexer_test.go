package indexer

import (
	"testing"

	"github.com/spf13/viper"
)

func TestGemList(t *testing.T) {
	viper.Set("gem_dir", "../../test/fixtures/gem_list")
	expected := []string{"../../test/fixtures/gem_list/a.gem", "../../test/fixtures/gem_list/another_@gem]2345.gem", "../../test/fixtures/gem_list/b.gem"}
	actual := gemList()
	if len(actual) != len(expected) {
		t.Errorf("Expected gem list with length %d. Received gem list with length %d", len(expected), len(actual))
	}
	for i, gem := range expected {
		if actual[i] != gem {
			t.Errorf("Expected %s at index %d. Received gem %s", gem, i, actual[i])
		}
	}
}