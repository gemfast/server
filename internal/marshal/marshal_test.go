package marshal

import (
	"encoding/hex"
	"testing"

	"github.com/gscho/gemfast/internal/marshal"
	"github.com/gscho/gemfast/internal/spec"
)

func TestDumpSpecs(t *testing.T) {
	rubyResult := "04085b075b0849220b646576697365063a064554553a1147656d3a3a56657273696f6e5b0649220a342e372e32063b005449220972756279063b00545b084922136d69786c69622d696e7374616c6c063b0054553b065b0649220c332e31322e3139063b005449220972756279063b0054"
	spec1 := spec.Spec{
		Name: "devise",
		Version: "4.7.2",
		OriginalPlatform: "ruby",
	}
	spec2 := spec.Spec{
		Name: "mixlib-install",
		Version: "3.12.19",
		OriginalPlatform: "ruby",
	}
	specs := []*spec.Spec{&spec1, &spec2}
	b := marshal.DumpSpecs(specs)
	if hex.EncodeToString(b) != rubyResult {
		t.Error("Dump does not match ruby result")
	}
}