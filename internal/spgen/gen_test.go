package spgen_test

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
)

func TestDump(t *testing.T) {
	out, err := spgen.EmitActionSel("../actionsel")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(out.Table.String())
}
