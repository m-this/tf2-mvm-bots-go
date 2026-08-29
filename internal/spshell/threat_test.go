package spshell_test

import (
	"errors"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spshell"
)

// threatScore is the Go side of the differential test: the same arithmetic as
// testdata/threat.sp, in float32 because SourcePawn has no other float.
func threatScore(distance, health, classID float32) float32 {
	rangeTerm := 1.0 - distance/2048.0
	if rangeTerm < 0.0 {
		rangeTerm = 0.0
	}

	hurt := 1.0 - health/300.0
	if hurt < 0.0 {
		hurt = 0.0
	}

	return rangeTerm*60.0 + hurt*30.0 + classID*1.5
}

func TestThreatScoreMatchesSourcePawn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input spshell.Triple
	}{
		{"point blank, full health, scout", spshell.Triple{0, 125, 0}},
		{"mid range, half health, soldier", spshell.Triple{1024, 150, 3}},
		{"out of range, clamps to zero", spshell.Triple{4096, 25, 8}},
		{"overhealed, hurt clamps to zero", spshell.Triple{512, 450, 2}},
		{"one unit short of the range cutoff", spshell.Triple{2047.9, 1, 1}},
		{"a distance with no exact float32", spshell.Triple{1234.5678, 199.99, 6}},
		{"negative distance, no clamp on the high side", spshell.Triple{-64, 300, 9}},
		{"all zero", spshell.Triple{0, 0, 0}},
	}

	toolchain := spshell.ForTest(t)

	inputs := make([]spshell.Triple, len(cases))
	for i, c := range cases {
		inputs[i] = c.input
	}

	got, err := toolchain.ScoreTriples(t.Context(), "testdata/threat.sp", inputs)
	if err != nil {
		t.Fatalf("running testdata/threat.sp: %v", err)
	}
	if len(got) != len(cases) {
		t.Fatalf("got %d scores for %d inputs", len(got), len(cases))
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := threatScore(c.input[0], c.input[1], c.input[2])
			if got[i] != want {
				t.Errorf("ThreatScore%v = %v under spshell, %v in Go", c.input, got[i], want)
			}
		})
	}
}

func TestToolchainFromEnvRequiresEveryPart(t *testing.T) {
	for _, missing := range []string{"SPCOMP", "SPSHELL", "SPINCLUDE"} {
		t.Run(missing, func(t *testing.T) {
			for _, name := range []string{"SPCOMP", "SPSHELL", "SPINCLUDE"} {
				t.Setenv(name, ".")
			}
			t.Setenv(missing, "")

			if _, err := spshell.ToolchainFromEnv(); !errors.Is(err, spshell.ErrNoToolchain) {
				t.Fatalf("with %s unset, got %v", missing, err)
			}
		})
	}
}
