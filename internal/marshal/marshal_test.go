package marshal

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/gscho/gemfast/internal/marshal"
	"github.com/gscho/gemfast/internal/spec"
)

func TestDumpSpecs(t *testing.T) {
	rubyResult := "04085b085b0849220b646576697365063a064554553a1147656d3a3a56657273696f6e5b0649220a342e372e32063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054553b065b0649220c332e31322e3139063b005449220972756279063b00545b084922076132063b0054553b065b0649220a302e332e30063b005449220972756279063b0054"

	spec1 := spec.Spec{
		Name:             "devise",
		Version:          "4.7.2",
		OriginalPlatform: "ruby",
	}
	spec2 := spec.Spec{
		Name:             "mixlib-install",
		Version:          "3.12.19",
		OriginalPlatform: "ruby",
	}
	spec3 := spec.Spec{
		Name:             "a2",
		Version:          "0.3.0",
		OriginalPlatform: "ruby",
	}
	specs := []*spec.Spec{&spec1, &spec2, &spec3}
	b := marshal.DumpSpecs(specs)
	t.Log(hex.EncodeToString(b))
	if hex.EncodeToString(b) != rubyResult {
		t.Error("Dump does not match ruby result")
	}
}

func TestDumpSpecsWithSameName(t *testing.T) {
	rubyResult := "04085b085b0849220b646576697365063a064554553a1147656d3a3a56657273696f6e5b0649220a342e372e32063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054553b065b0649220c332e31322e3139063b005449220972756279063b00545b0849220b646576697365063b0054553b065b0649220a342e372e31063b005449220972756279063b0054"
	spec1 := spec.Spec{
		Name:             "devise",
		Version:          "4.7.2",
		OriginalPlatform: "ruby",
	}
	spec2 := spec.Spec{
		Name:             "mixlib-install",
		Version:          "3.12.19",
		OriginalPlatform: "ruby",
	}
	spec3 := spec.Spec{
		Name:             "devise",
		Version:          "4.7.1",
		OriginalPlatform: "ruby",
	}
	specs := []*spec.Spec{&spec1, &spec2, &spec3}
	b := marshal.DumpSpecs(specs)
	t.Log(hex.EncodeToString(b))
	if hex.EncodeToString(b) != rubyResult {
		t.Error("Dump does not match ruby result")
	}
}

func TestLoadSpecs(t *testing.T) {
	rubyResult := "04085b085b0849220b646576697365063a064554553a1147656d3a3a56657273696f6e5b0649220a342e372e32063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054553b065b0649220c332e31322e3139063b005449220972756279063b00545b084922076132063b0054553b065b0649220a302e332e30063b005449220972756279063b0054"
	rubyBytes, _ := hex.DecodeString(rubyResult)
	buff := bytes.NewBuffer(rubyBytes)
	specs := marshal.LoadSpecs(buff)
	t.Log("Loaded specs length: ", len(specs))
	if len(specs) != 3 {
		t.Error("Loaded specs length does not match ruby")
	}
	if specs[0].Name != "devise" {
		t.Errorf("Expected 'devise', got: %s", specs[0].Name)
	}
	if specs[0].Version != "4.7.2" {
		t.Errorf("Expected '4.7.2', got: %s", specs[0].Version)
	}
	if specs[0].OriginalPlatform != "ruby" {
		t.Errorf("Expected 'ruby', got: %s", specs[0].OriginalPlatform)
	}
	if specs[1].Name != "mixlib-install" {
		t.Errorf("Expected 'mixlib-install', got: %s", specs[1].Name)
	}
	if specs[1].Version != "3.12.19" {
		t.Errorf("Expected '3.12.19', got: %s", specs[1].Version)
	}
	if specs[1].OriginalPlatform != "ruby" {
		t.Errorf("Expected 'ruby', got: %s", specs[1].OriginalPlatform)
	}
	if specs[2].Name != "a2" {
		t.Errorf("Expected 'a2', got: %s", specs[2].Name)
	}
	if specs[2].Version != "0.3.0" {
		t.Errorf("Expected '0.3.0', got: %s", specs[2].Version)
	}
	if specs[2].OriginalPlatform != "ruby" {
		t.Errorf("Expected 'ruby', got: %s", specs[2].OriginalPlatform)
	}
}
