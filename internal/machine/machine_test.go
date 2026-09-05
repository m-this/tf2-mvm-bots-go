package machine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParsesProcFiles(t *testing.T) {
	mem, err := parseMemAvailable("MemTotal:       16000000 kB\nMemFree:         1000000 kB\nMemAvailable:    8000000 kB\n")
	if err != nil || mem != 8000000 {
		t.Fatalf("MemAvailable %d, %v", mem, err)
	}
	load, err := parseLoad("1.25 0.80 0.60 2/900 12345\n")
	if err != nil || load != 1.25 {
		t.Fatalf("load %v, %v", load, err)
	}
	if _, err := parseMemAvailable("MemTotal: 1 kB\n"); err == nil {
		t.Error("a meminfo with no MemAvailable was accepted")
	}
}

func TestChecksumsNameEverySharedObject(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"a.ext.so": "aaa", "b.ext.so": "bbb", "notes.txt": "x"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := checksums(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || len(got["a.ext.so"]) != 12 || got["a.ext.so"] == got["b.ext.so"] {
		t.Errorf("checksums %v", got)
	}
	none, err := checksums(filepath.Join(dir, "missing"))
	if err != nil || len(none) != 0 {
		t.Errorf("a missing directory gave %v, %v", none, err)
	}
}

func TestComparableRefusesWhatDiffers(t *testing.T) {
	same := Machine{Host: "bed", MemAvailableKB: 4 << 20, Load1m: 1.0, Extensions: map[string]string{"x.so": "abc"}}
	cases := []struct {
		name  string
		other Machine
		ok    bool
	}{
		{"the same machine", same, true},
		{"a little more load", Machine{Host: "bed", MemAvailableKB: 4 << 20, Load1m: 2.5, Extensions: same.Extensions}, true},
		{"another host", Machine{Host: "laptop", MemAvailableKB: 4 << 20, Load1m: 1.0, Extensions: same.Extensions}, false},
		{"another extension build", Machine{Host: "bed", MemAvailableKB: 4 << 20, Load1m: 1.0, Extensions: map[string]string{"x.so": "def"}}, false},
		{"an extension missing", Machine{Host: "bed", MemAvailableKB: 4 << 20, Load1m: 1.0, Extensions: map[string]string{}}, false},
		{"load far apart", Machine{Host: "bed", MemAvailableKB: 4 << 20, Load1m: 3.5, Extensions: same.Extensions}, false},
		{"under the memory floor", Machine{Host: "bed", MemAvailableKB: 512 << 10, Load1m: 1.0, Extensions: same.Extensions}, false},
	}
	for _, c := range cases {
		err := Comparable([]Machine{same}, []Machine{c.other})
		if c.ok && err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
		}
		if !c.ok && !errors.Is(err, ErrNotComparable) {
			t.Errorf("%s: accepted, or the wrong error: %v", c.name, err)
		}
	}
	if err := Comparable(nil, nil); err != nil {
		t.Errorf("nothing to compare was refused: %v", err)
	}
}
