package address_test

import (
	"slices"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/body/address"
	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
	"github.com/m-this/tf2-mvm-bots-go/spshell"
)

var cells = []int32{0, 1, -1, 0x7FFFFFFF, -0x7FFFF000}

// goCells runs the cases address_probe.sp runs and prints what it prints.
func goCells() []int32 {
	thrown := int32(0)
	defer engine.InstallManages(engine.ManageCalls{
		ThrowError: func(string, []any) { thrown = 1 },
	})()

	var out []int32
	emitAddress := func(a engine.Address) {
		out = append(out, thrown, int32(a), int32(a>>32)) //nolint:gosec // G115: the halves of an int64, as the probe prints them
		thrown = 0
	}
	emitCell := func(c int32) {
		out = append(out, thrown, c)
		thrown = 0
	}
	for _, c := range cells {
		emitAddress(address.Widen(c))
		emitCell(address.Narrow(address.Widen(c)))
		emitCell(address.Narrow(engine.Address(c)))
	}
	emitCell(address.Narrow(engine.Address(1) << 32))
	emitCell(address.Narrow(engine.Address(-1)<<32 | 5))
	emitCell(address.Narrow(engine.Address(2)<<32 | 0x80001000))
	return out
}

func TestGoNarrowsBothFormsAndThrowsOnTheRest(t *testing.T) {
	got := goCells()
	for i, c := range cells {
		widened := got[i*7 : i*7+7]
		if widened[0] != 0 || widened[1] != c || widened[2] != 0 {
			t.Errorf("Widen(%d) = thrown %d, low %d, high %d; want zero-extended", c, widened[0], widened[1], widened[2])
		}
		if widened[3] != 0 || widened[4] != c || widened[5] != 0 || widened[6] != c {
			t.Errorf("Narrow of %d: %v; want the cell back from both forms, no throw", c, widened[3:])
		}
	}
	invalid := got[len(cells)*7:]
	for i := 0; i < len(invalid); i += 2 {
		if invalid[i] != 1 {
			t.Errorf("Narrow of an Address that fits no cell did not throw: %v", invalid)
		}
	}
}

func TestGeneratedAddressAgreesWithTheGoItCameFrom(t *testing.T) {
	tc := spshell.ForTest(t)

	generated, err := body.Generate("../../..")
	if err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	got, err := tc.Run(t.Context(), "testdata/address_probe.sp", map[string]string{
		"address.sp": string(generated["sourcepawn/address.sp"]),
	})
	if err != nil {
		t.Fatalf("running the probe under spshell: %v", err)
	}
	if want := goCells(); !slices.Equal(got, want) {
		t.Errorf("probe printed\n%v\nGo printed\n%v", got, want)
	}
}
