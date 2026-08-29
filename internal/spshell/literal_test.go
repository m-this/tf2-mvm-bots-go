package spshell_test

import (
	"math"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
)

/*
	FuzzFloatLiteralSurvivesTheCompiler

Everything generated that holds a number rests on this. A literal spcomp reads
back as a different float32 is a silent wrong answer in every generated function
that carries one, and it would not show up as a compile error.

The plugin echoes the inputs rather than computing with them, so a difference is
the compiler reading what Go wrote and nothing else. The comparison is on bits,
which is what makes a negative zero or a denormal a failure rather than a value
that prints the same.
*/
func FuzzFloatLiteralSurvivesTheCompiler(f *testing.F) {
	// The corpus is the shapes shortest-decimal is most likely to get wrong:
	// the denormal floor, the normal floor and ceiling, the ends of exact
	// integer range, and a negative zero, which has a bit pattern of its own.
	for _, seed := range []uint32{
		0x00000000, 0x80000000, 0x00000001, 0x007fffff, 0x00800000,
		0x7f7fffff, 0x3f800000, 0x4b7fffff, 0x4b800000, 0x3eaaaaab,
	} {
		f.Add(seed)
	}

	toolchain := spshell.ForTest(f)

	f.Fuzz(func(t *testing.T, bits uint32) {
		v := math.Float32frombits(bits)
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			// SourcePawn has no literal for either, so the generator cannot
			// emit one and there is nothing here to hold the compiler to.
			t.Skip("not a finite float32")
		}

		got, err := toolchain.ScoreTriples(t.Context(), "testdata/echo.sp", []spshell.Triple{{v, v, v}})
		if err != nil {
			t.Fatalf("echoing %v (%#08x): %v", v, bits, err)
		}
		for i, back := range got {
			if math.Float32bits(back) != bits {
				t.Fatalf("cell %d: wrote %v as a literal, spcomp read back %#08x, want %#08x",
					i, v, math.Float32bits(back), bits)
			}
		}
	})
}
