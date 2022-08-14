package marshal

import (
	"bufio"
	"bytes"
	"encoding/hex"
	// "os"
	"testing"

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
	b := DumpSpecs(specs)
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
	b := DumpSpecs(specs)
	if hex.EncodeToString(b) != rubyResult {
		t.Error("Dump does not match ruby result")
	}
}

func TestLoadSpecs(t *testing.T) {
	rubyResult := "04085b085b0849220b646576697365063a064554553a1147656d3a3a56657273696f6e5b0649220a342e372e32063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054553b065b0649220c332e31322e3139063b005449220972756279063b00545b084922076132063b0054553b065b0649220a302e332e30063b005449220972756279063b0054"
	rubyBytes, _ := hex.DecodeString(rubyResult)
	buff := bytes.NewBuffer(rubyBytes)
	specs := LoadSpecs(buff)
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

func TestReadInt(t *testing.T) {
	rubyResult := "035ca002"
	rubyBytes, _ := hex.DecodeString(rubyResult)
	buff := bytes.NewBuffer(rubyBytes)
	r := bufio.NewReader(buff)
	l, _ := readInt(r)
	if l != 172124 {
		t.Errorf("Error, expected '172124', got: %b ", l)
	}
}

func TestDumpGemspecGemfastStyle(t *testing.T) {
	rubyResult := "04086f3a1747656d3a3a53706563696669636174696f6e143a0a406e616d6549220c636f6d706f7365063a0645543a0d4076657273696f6e553a1147656d3a3a56657273696f6e5b0649220a302e312e30063b07543a0d4073756d6d61727949223a577269746520612073686f72742073756d6d6172792c2062656361757365205275627947656d73207265717569726573206f6e652e063b07543a1b4072657175697265645f727562795f76657273696f6e553a1547656d3a3a526571756972656d656e745b065b065b074922073e3d063b0754553b095b0649220a322e362e30063b07543a1f4072657175697265645f7275627967656d735f76657273696f6e553a1547656d3a3a526571756972656d656e745b065b065b074922073e3d063b0754553b095b0649220a332e332e33063b07543a17406f726967696e616c5f706c6174666f726d49220972756279063b07543a0b40656d61696c5b0649221f677265672e632e7363686f6669656c6440676d61696c2e636f6d063b07543a0d40617574686f72735b06492216477265676f7279205363686f6669656c64063b07543a11406465736372697074696f6e49223457726974652061206c6f6e676572206465736372697074696f6e206f722064656c6574652074686973206c696e652e063b07543a0e40686f6d657061676549222d68747470733a2f2f6769746875622e636f6d2f677363686f2f686162697461742d636f6d706f7365063b07543a0e406c6963656e7365735b064922084d4954063b07543a1340726571756972655f70617468735b064922086c6962063b07543a1b4073706563696669636174696f6e5f76657273696f6e69093a1240646570656e64656e636965735b066f3a1447656d3a3a446570656e64656e63790a3b0649220e66696c652d7461696c063b07543a1140726571756972656d656e74553a1547656d3a3a526571756972656d656e745b065b065b074922077e3e063b0754553b095b06492208312e32063b07543a0a40747970653a0c72756e74696d653a104070726572656c65617365463a1a4076657273696f6e5f726571756972656d656e7473401a3a16407275627967656d735f76657273696f6e49220a332e332e33063b0754"
	rubyBytes, _ := hex.DecodeString(rubyResult)
	m := spec.GemMetadata{
		Name:     "compose",
		Platform: "ruby",
		Version: struct {
			Version string `yaml:"version"`
		}{
			Version: "0.1.0",
		},
		Authors:         []string{"Gregory Schofield", "Skyler Layne"},
		Email:           []string{"greg.c.schofield@gmail.com", "greg.schofield@indellient.com"},
		Summary:         "Write a short summary, because RubyGems requires one.",
		Description:     "Write a longer description or delete this line.",
		Homepage:        "https://github.com/gscho/habitat-compose",
		SpecVersion:     4,
		RequirePaths:    []string{"lib", "bin"},
		Licenses:        []string{"MIT", "unlicense"},
		RubygemsVersion: "3.3.3",
	}
	gs := DumpGemspecGemfast(m)
	for i, b := range rubyBytes {
		if i >= len(gs) {
			// t.Errorf("%x", b)
			t.Fatalf("Previous byte was '%x' or '%s' or '%d'.\nNext byte would have been '%x', aka '%s' or '%d'", rubyBytes[i-1], string(rubyBytes[i-1]), rubyBytes[i-1], b, string(b), b)
			// os.Exit(1)
		}
		if gs[i] != b {
			t.Fatalf("Error, expected '%x' or '%s' or '%d'.\nReceived '%x' or '%s' or '%d' at index %d", b, string(b), b, gs[i], string(gs[i]), gs[i], i)
			// os.Exit(1)
		}
	}
}
