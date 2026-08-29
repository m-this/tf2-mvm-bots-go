package tables_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/tables"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// TestGolden pins the generated text. The round-trip tests say the names are
// right; this says a change to the layout is deliberate.
func TestGolden(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		got  []byte
	}{
		{"features.sp", tables.SourcePawnFeatures()},
		{"arms.go", tables.GoFeatureArms("arms")},
		{"wave_write.sp", tables.SourcePawnWaveWriter()},
		{"wave.go", tables.GoWaveParser("waveline")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join("testdata", tc.name+".golden")
			if *update {
				if err := os.WriteFile(path, tc.got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run go test -update)", err)
			}
			if !bytes.Equal(tc.got, want) {
				t.Errorf("%s differs from the golden file", tc.name)
			}
		})
	}
}
