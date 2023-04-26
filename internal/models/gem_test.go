package models

import (
	"testing"
)

func TestGemFromGemParameter(t *testing.T) {
	name := "activesupport-7.0.4.3"
	g := GemFromGemParameter(name)
	if g.Name != "activesupport" {
		t.Errorf("expected gem named activesupport")
	}
	if g.Number != "7.0.4.3" {
		t.Errorf("expected gem version 7.0.4.3")
	}

	name = "activerecord-oracle_enhanced-adapter-1.1.8"
	g = GemFromGemParameter(name)
	if g.Name != "activerecord-oracle_enhanced-adapter" {
		t.Errorf("expected gem named activerecord-oracle_enhanced-adapter")
	}
	if g.Number != "1.1.8" {
		t.Errorf("expected gem version 1.1.8")
	}

	name = ""
	g = GemFromGemParameter(name)
	if g.Name != "" {
		t.Errorf("expected gem name empty")
	}
	if g.Number != "" {
		t.Errorf("expected gem version empty")
	}
}
