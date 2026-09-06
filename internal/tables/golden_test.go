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

/*
	TestGolden pins the generated text a diff would otherwise never show

The round-trip tests say the names are right; this says a change to the layout
is deliberate. Only the two that land in gen/, which is gitignored: the
SourcePawn tables are committed where the plugin includes them, so a change to
features.sp or wave_write.sp already reads as a diff there and
TestAdoptedFilesMatchTheGenerator is what holds them to the generator. A golden
beside those would be the same bytes written down twice.
*/
func TestGolden(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		got  []byte
	}{
		{"arms.go", tables.GoFeatureArms("arms")},
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
