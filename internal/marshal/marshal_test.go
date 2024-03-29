package marshal

import (
	"bufio"
	"bytes"
	"encoding/hex"

	// "strings"

	// "os"
	"testing"

	"github.com/gemfast/server/internal/db"
	// "github.com/gemfast/server/internal/spec"
)

// func TestDumpSpecsWithMultiPlatforms(t *testing.T) {
// 	rubyResult := "04085b0a5b0849220877646d063a064554553a1147656d3a3a56657273696f6e5b0649220a302e302e31063b005449220972756279063b00545b0849220877646d063b0054553b065b0649220a302e302e32063b005449220b6d696e673332063b00545b0849220877646d063b0054400e4922107838362d6d696e67773332063b00545b0849220877646d063b0054553b065b0649220a302e312e31063b005449220972756279063b00545b0849220877646d063b0054553b065b0649220a302e312e30063b005449220972756279063b0054"
// 	rubyBytes, _ := hex.DecodeString(rubyResult)
// 	testSpecs := [][]string{
// 		[]string{"wdm", "0.0.1", "ruby"},
// 		[]string{"wdm", "0.0.2", "ming32"},
// 		[]string{"wdm", "0.0.2", "x86-mingw32"},
// 		[]string{"wdm", "0.1.1", "ruby"},
// 		[]string{"wdm", "0.1.0", "ruby"},
// 	}

// 	var specs []*spec.Spec
// 	for _, ts := range testSpecs {
// 		specs = append(specs, &spec.Spec{
// 			Name:             ts[0],
// 			Version:          ts[1],
// 			OriginalPlatform: ts[2],
// 		})
// 	}
// 	dump := DumpSpecs(specs)
// 	for i, b := range rubyBytes {
// 		if i >= len(dump) {
// 			// t.Errorf("%x", b)
// 			t.Fatalf("Previous byte was '%x' or '%s' or '%d'.\nNext byte would have been '%x', aka '%s' or '%d'", rubyBytes[i-1], string(rubyBytes[i-1]), rubyBytes[i-1], b, string(b), b)
// 			// os.Exit(1)
// 		}
// 		if dump[i] != b {
// 			t.Fatalf("Error, expected '%x' or '%s' or '%d'.\nReceived '%x' or '%s' or '%d' at index %d", b, string(b), b, dump[i], string(dump[i]), dump[i], i)
// 			// os.Exit(1)
// 		}
// 	}
// }

// func TestDumpSpecs(t *testing.T) {
// 	rubyResult := "04085b135b0849220f636865662d7574696c73063a064554553a1147656d3a3a56657273696f6e5b0649220c31372e31302e30063b005449220972756279063b00545b08492214636f6e63757272656e742d72756279063b0054553b065b0649220b312e312e3130063b005449220972756279063b00545b0849220b646576697365063b0054553b065b0649220a342e372e31063b005449220972756279063b00545b0849220b646576697365063b0054553b065b0649220a342e372e32063b005449220972756279063b00545b08492214666163746f72795f6d616e61676572063b0054553b065b0649220a302e332e30063b005449220972756279063b00545b0849220e6c6974746572626f78063b0054553b065b0649220a302e342e30063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054553b065b0649220a332e302e30063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054553b065b0649220c332e31322e3139063b005449220972756279063b00545b084922146d69786c69622d7368656c6c6f7574063b0054553b065b0649220a332e322e37063b005449220972756279063b00545b084922166d69786c69622d76657273696f6e696e67063b0054553b065b0649220a312e302e30063b005449220972756279063b00545b084922166d69786c69622d76657273696f6e696e67063b0054553b065b0649220b312e322e3132063b005449220972756279063b00545b084922077067063b0054553b065b0649220a312e342e33063b005449220972756279063b00545b0849220974686f72063b0054553b065b0649220a312e322e31063b005449220972756279063b00545b0849220d77696e3332617069063b0054553b065b0649220a302e312e30063b005449220972756279063b0054"

// 	testSpecs := []string{
// 		"chef-utils-17.10.0.gem",
// 		"concurrent-ruby-1.1.10.gem",
// 		"devise-4.7.1.gem",
// 		"devise-4.7.2.gem",
// 		"factory_manager-0.3.0.gem",
// 		"litterbox-0.4.0.gem",
// 		"mixlib-install-3.0.0.gem",
// 		"mixlib-install-3.12.19.gem",
// 		"mixlib-shellout-3.2.7.gem",
// 		"mixlib-versioning-1.0.0.gem",
// 		"mixlib-versioning-1.2.12.gem",
// 		"pg-1.4.3.gem",
// 		"thor-1.2.1.gem",
// 		"win32api-0.1.0.gem",
// 	}

// 	var specs []*spec.Spec
// 	for _, ts := range testSpecs {
// 		chunks := strings.Split(ts, "-")
// 		version := strings.Split(chunks[len(chunks)-1], ".gem")[0]
// 		chunks = chunks[:len(chunks)-1]
// 		name := strings.Join(chunks, "-")
// 		specs = append(specs, &spec.Spec{
// 			Name:             name,
// 			Version:          version,
// 			OriginalPlatform: "ruby",
// 		})
// 	}

// 	b := DumpSpecs(specs)
// 	if hex.EncodeToString(b) != rubyResult {
// 		t.Error("Dump does not match ruby result")
// 	}
// }

// func TestDumpSpecsWithCaching(t *testing.T) {
// 	rubyResult := "04085b075b084922136d69786c69622d696e7374616c6c063a064554553a1147656d3a3a56657273696f6e5b0649220c332e31322e3139063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054400849220972756279063b0054"
// 	rubyBytes, _ := hex.DecodeString(rubyResult)
// 	spec1 := spec.Spec{
// 		Name:             "mixlib-install",
// 		Version:          "3.12.19",
// 		OriginalPlatform: "ruby",
// 	}
// 	spec2 := spec.Spec{
// 		Name:             "mixlib-install",
// 		Version:          "3.12.19",
// 		OriginalPlatform: "ruby",
// 	}

// 	specs := []*spec.Spec{&spec1, &spec2}
// 	dump := DumpSpecs(specs)
// 	for i, b := range rubyBytes {
// 		if i >= len(dump) {
// 			// t.Errorf("%x", b)
// 			t.Fatalf("Previous byte was '%x' or '%s' or '%d'.\nNext byte would have been '%x', aka '%s' or '%d'", rubyBytes[i-1], string(rubyBytes[i-1]), rubyBytes[i-1], b, string(b), b)
// 			// os.Exit(1)
// 		}
// 		if dump[i] != b {
// 			t.Fatalf("Error, expected '%x' or '%s' or '%d'.\nReceived '%x' or '%s' or '%d' at index %d", b, string(b), b, dump[i], string(dump[i]), dump[i], i)
// 			// os.Exit(1)
// 		}
// 	}
// }

// func TestDumpSpecsWithSameName(t *testing.T) {
// 	rubyResult := "04085b085b0849220b646576697365063a064554553a1147656d3a3a56657273696f6e5b0649220a342e372e32063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054553b065b0649220c332e31322e3139063b005449220972756279063b00545b0849220b646576697365063b0054553b065b0649220a342e372e31063b005449220972756279063b0054"
// 	spec1 := spec.Spec{
// 		Name:             "devise",
// 		Version:          "4.7.2",
// 		OriginalPlatform: "ruby",
// 	}
// 	spec2 := spec.Spec{
// 		Name:             "mixlib-install",
// 		Version:          "3.12.19",
// 		OriginalPlatform: "ruby",
// 	}
// 	spec3 := spec.Spec{
// 		Name:             "devise",
// 		Version:          "4.7.1",
// 		OriginalPlatform: "ruby",
// 	}
// 	specs := []*spec.Spec{&spec1, &spec2, &spec3}
// 	b := DumpSpecs(specs)
// 	if hex.EncodeToString(b) != rubyResult {
// 		t.Error("Dump does not match ruby result")
// 	}
// }

func TestLoadSpecsWithUnderscore(t *testing.T) {
	rubyResult := "04085b075b084922065f063a064554553a1147656d3a3a56657273696f6e5b06492208312e30063b005449220972756279063b00545b084007553b065b06492208312e31063b0054400b"
	rubyBytes, _ := hex.DecodeString(rubyResult)
	buff := bytes.NewBuffer(rubyBytes)
	specs := LoadSpecs(buff)
	if len(specs) != 2 {
		t.Error("Loaded specs length does not match ruby")
	}
	if specs[0].Name != "_" {
		t.Errorf("Expected '_', got: %s", specs[0].Name)
	}
	if specs[0].Version != "1.0" {
		t.Errorf("Expected '1.0', got: %s", specs[0].Version)
	}
	if specs[0].OriginalPlatform != "ruby" {
		t.Errorf("Expected 'ruby', got: %s", specs[0].OriginalPlatform)
	}
	if specs[1].Name != "_" {
		t.Errorf("Expected '_', got: %s", specs[1].Name)
	}
	if specs[1].Version != "1.1" {
		t.Errorf("Expected '1.1', got: %s", specs[1].Version)
	}
	if specs[1].OriginalPlatform != "ruby" {
		t.Errorf("Expected 'ruby', got: %s", specs[1].OriginalPlatform)
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

func TestDumpBundlerDeps(t *testing.T) {
	rubyResult := "04085b077b093a096e616d654922136d69786c69622d696e7374616c6c063a0645543a0b6e756d62657249220c332e31322e3139063b06543a0d706c6174666f726d49220972756279063b06543a11646570656e64656e636965735b085b074922146d69786c69622d7368656c6c6f7574063b06544922093e3d2030063b06545b074922166d69786c69622d76657273696f6e696e67063b06544922093e3d2030063b06545b0749220974686f72063b06544922093e3d2030063b06547b093b004922136d69786c69622d696e7374616c6c063b06543b0749220c332e31322e3139063b06543b0849220972756279063b06543b095b085b074922146d69786c69622d7368656c6c6f7574063b06544922093e3d2030063b06545b074922166d69786c69622d76657273696f6e696e67063b06544922093e3d2030063b06545b0749220974686f72063b06544922093e3d2030063b0654"
	rubyBytes, _ := hex.DecodeString(rubyResult)
	var deps []*db.Gem
	d1 := &db.Gem{
		Name:         "mixlib-install",
		Number:       "3.12.19",
		Platform:     "ruby",
		Dependencies: []db.GemDependency{{Name: "mixlib-shellout", Type: ":runtime", VersionConstraints: ">= 0"}, {Name: "mixlib-versioning", Type: ":runtime", VersionConstraints: ">= 0"}, {Name: "thor", Type: ":runtime", VersionConstraints: ">= 0"}},
	}
	deps = append(deps, d1)
	d2 := &db.Gem{
		Name:         "mixlib-install",
		Number:       "3.12.19",
		Platform:     "ruby",
		Dependencies: []db.GemDependency{{Name: "mixlib-shellout", Type: ":runtime", VersionConstraints: ">= 0"}, {Name: "mixlib-versioning", Type: ":runtime", VersionConstraints: ">= 0"}, {Name: "thor", Type: ":runtime", VersionConstraints: ">= 0"}},
	}
	deps = append(deps, d2)
	bd, _ := DumpBundlerDeps(deps)
	for i, b := range rubyBytes {
		if i >= len(bd) {
			// t.Errorf("%x", b)
			t.Errorf("Previous byte was '%x' or '%s' or '%d'.\nNext byte would have been '%x', aka '%s' or '%d'", rubyBytes[i-1], string(rubyBytes[i-1]), rubyBytes[i-1], b, string(b), b)
			// os.Exit(1)
		}
		if bd[i] != b {
			t.Errorf("Error, expected '%x' or '%s' or '%d'.\nReceived '%x' or '%s' or '%d' at index %d", b, string(b), b, bd[i], string(bd[i]), bd[i], i)
			// os.Exit(1)
		}
	}
}

// func TestDumpGemspecGemfast(t *testing.T) {
// 	rubyResult := "0408753a1747656d3a3a53706563696669636174696f6e02060204085b1849220a332e342e33063a0645546909492211616374696f6e6d61696c6572063b0054553a1147656d3a3a56657273696f6e5b0649220c372e302e342e33063b005449753a0954696d650de0ca1ec000000000063a097a6f6e65492208555443063b004649223e456d61696c20636f6d706f736974696f6e20616e642064656c6976657279206672616d65776f726b202870617274206f66205261696c73292e063b0054553a1547656d3a3a526571756972656d656e745b065b065b074922073e3d063b0054553b065b0649220a322e362e30063b0054553b095b065b065b074012553b065b0649220630063b0046305b00492200063b005449221b6461766964406c6f75647468696e6b696e672e636f6d063b00545b0649221d4461766964204865696e656d656965722048616e73736f6e063b005449220196456d61696c206f6e205261696c732e20436f6d706f73652c2064656c697665722c20616e64207465737420656d61696c73207573696e67207468652066616d696c69617220636f6e74726f6c6c65722f76696577207061747465726e2e2046697273742d636c61737320737570706f727420666f72206d756c74697061727420656d61696c20616e64206174746163686d656e74732e063b005449221c68747470733a2f2f727562796f6e7261696c732e6f7267063b00545449220972756279063b00545b007b00"
// 	rubyBytes, _ := hex.DecodeString(rubyResult)
// 	m := spec.GemMetadata{
// 		Name:     "actionmailer",
// 		Platform: "ruby",
// 		Version: struct {
// 			Version string `yaml:"version"`
// 		}{
// 			Version: "7.0.4.3",
// 		},
// 		Authors:         []string{"David Heinemeier Hansson"},
// 		Email:           []string{"david@loudthinking.com"},
// 		Summary:         "Email composition and delivery framework (part of Rails).",
// 		Description:     "Email on Rails. Compose, deliver, and test emails using the familiar controller/view pattern. First-class support for multipart email and attachments.",
// 		Homepage:        "https://rubyonrails.org",
// 		SpecVersion:     4,
// 		RequirePaths:    []string{"lib"},
// 		// Licenses:        []string{"MIT", "unlicense"},
// 		RubygemsVersion: "3.4.3",
// 	}
// 	gs := DumpGemspecGemfast(m)
// 	for i, b := range rubyBytes {
// 		if i >= len(gs) {
// 			// t.Errorf("%x", b)
// 			t.Fatalf("Previous byte was '%x' or '%s' or '%d'.\nNext byte would have been '%x', aka '%s' or '%d'", rubyBytes[i-1], string(rubyBytes[i-1]), rubyBytes[i-1], b, string(b), b)
// 			// os.Exit(1)
// 		}
// 		if gs[i] != b {
// 			t.Fatalf("Error, expected '%x' or '%s' or '%d'.\nReceived '%x' or '%s' or '%d' at index %d", b, string(b), b, gs[i], string(gs[i]), gs[i], i)
// 			// os.Exit(1)
// 		}
// 	}
// }
