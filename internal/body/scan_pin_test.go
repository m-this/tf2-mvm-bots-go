package body_test

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var updateScanCells = flag.Bool("update-scan-cells", false, "rewrite testdata/scan_cells.golden from the Go scans")

/*
	TestScanTracesArePinned

The differential test proves the generated SourcePawn does what the Go does. It
cannot say the Go still does what it did, because both sides move together. This
pins the Go side: every answer and every engine call, in order, over the canned
world. The collapse of the nine scans into one loop (mvm-z83.92) is allowed to
change nothing here.
*/
func TestScanTracesArePinned(t *testing.T) {
	got := goScanCells(cannedWorld())
	golden := "testdata/scan_cells.golden"
	if *updateScanCells {
		if err := os.WriteFile(golden, []byte(formatCells(got)), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	body, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v: run go test ./internal/body -run TestScanTracesArePinned -update-scan-cells", err)
	}
	want, err := parseCells(string(body))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(want, got) {
		i := 0
		for i < len(want) && i < len(got) && want[i] == got[i] {
			i++
		}
		t.Fatalf("the scans' answers or engine calls moved at cell %d of %d: want %v, got %v",
			i, len(want), window(want, i), window(got, i))
	}
}

func formatCells(cells []int32) string {
	var b strings.Builder
	for i, c := range cells {
		if i > 0 {
			if i%16 == 0 {
				b.WriteByte('\n')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(strconv.Itoa(int(c)))
	}
	b.WriteByte('\n')
	return b.String()
}

func parseCells(text string) ([]int32, error) {
	fields := strings.Fields(text)
	cells := make([]int32, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.ParseInt(f, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("scan_cells.golden: %w", err)
		}
		cells = append(cells, int32(v))
	}
	return cells, nil
}

func window(cells []int32, at int) []int32 {
	lo, hi := max(at-4, 0), min(at+8, len(cells))
	return cells[lo:hi]
}
