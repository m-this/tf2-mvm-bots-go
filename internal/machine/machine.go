/*
Package machine is what a run was played on, recorded so that two arms can be
refused a comparison when they were not played on the same thing.

The variance that ruined measurements was never statistical: one run was
paging, one had a live .so copied under it, two were unequal machine load. A
noise floor measured across two different machines measures the machines.
*/
package machine

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Machine is the record. It goes into the run record's first line, so a file
// found later says what it was played on.
type Machine struct {
	Host string `json:"host"`
	// MemAvailableKB is /proc/meminfo's MemAvailable at the start of the
	// attempt: what the server could take before the machine pages.
	MemAvailableKB int64 `json:"mem_available_kb"`
	// Load1m is the one minute load average at the start of the attempt.
	Load1m float64 `json:"load_1m"`
	// Extensions is the SourceMod extensions the server was built with, by
	// file name, each as the first twelve hex digits of its SHA-256.
	Extensions map[string]string `json:"extensions"`
}

/*
The two thresholds, and the bugs behind them.

MemAvailableFloorKB is one gibibyte. mvm-ibb was a run that paged, and a server
that pages has a frame time that belongs to the disk. LoadGapMax is two: mvm-8ws
and mvm-a8g were arms played under unequal machine load, and two whole cores of
difference is well past what a bed's own noise moves between attempts.
*/
const (
	MemAvailableFloorKB int64   = 1 << 20
	LoadGapMax          float64 = 2.0
)

// ErrNotComparable says two arms were not played on the same thing.
var ErrNotComparable = errors.New("the arms were not played on the same machine")

// Snapshot reads the machine now, and checksums the extensions under dir.
func Snapshot(extensionsDir string) (Machine, error) {
	host, err := os.Hostname()
	if err != nil {
		return Machine{}, err
	}
	meminfo, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return Machine{}, err
	}
	mem, err := parseMemAvailable(string(meminfo))
	if err != nil {
		return Machine{}, err
	}
	loadavg, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return Machine{}, err
	}
	load, err := parseLoad(string(loadavg))
	if err != nil {
		return Machine{}, err
	}
	extensions, err := checksums(extensionsDir)
	if err != nil {
		return Machine{}, err
	}
	return Machine{Host: host, MemAvailableKB: mem, Load1m: load, Extensions: extensions}, nil
}

func parseMemAvailable(meminfo string) (int64, error) {
	for line := range strings.SplitSeq(meminfo, "\n") {
		if !strings.HasPrefix(line, "MemAvailable:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}
		return strconv.ParseInt(fields[1], 10, 64)
	}
	return 0, errors.New("/proc/meminfo has no MemAvailable line")
}

func parseLoad(loadavg string) (float64, error) {
	fields := strings.Fields(loadavg)
	if len(fields) == 0 {
		return 0, errors.New("/proc/loadavg is empty")
	}
	return strconv.ParseFloat(fields[0], 64)
}

// checksums is every .so under dir, by name. A directory that is not there is
// a server built without extensions, which is recorded as none rather than
// refused: the comparison is what refuses.
func checksums(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".so" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name())) //nolint:gosec // the directory is the plugin's own build
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(body)
		out[entry.Name()] = hex.EncodeToString(sum[:6])
	}
	return out, nil
}

/*
Comparable says whether the arms may be read against each other, and if not,
what differs. Every machine is held against the first: the host and the
extensions must be the same, no attempt may have started under the memory
floor, and no two attempts may be further apart in load than LoadGapMax.
*/
func Comparable(arms ...[]Machine) error {
	var all []Machine
	for _, arm := range arms {
		all = append(all, arm...)
	}
	if len(all) == 0 {
		return nil
	}
	first := all[0]
	for _, m := range all[1:] {
		if m.Host != first.Host {
			return fmt.Errorf("%w: %s and %s", ErrNotComparable, first.Host, m.Host)
		}
		if diff := extensionsDiffer(first.Extensions, m.Extensions); diff != "" {
			return fmt.Errorf("%w: the extensions differ, %s", ErrNotComparable, diff)
		}
		if math.Abs(m.Load1m-first.Load1m) > LoadGapMax {
			return fmt.Errorf("%w: load was %.2f for one attempt and %.2f for another, more than %.0f apart",
				ErrNotComparable, first.Load1m, m.Load1m, LoadGapMax)
		}
	}
	for _, m := range all {
		if m.MemAvailableKB < MemAvailableFloorKB {
			return fmt.Errorf("%w: an attempt started with %d MiB available, under the %d MiB floor, so the server may have paged",
				ErrNotComparable, m.MemAvailableKB>>10, MemAvailableFloorKB>>10)
		}
	}
	return nil
}

func extensionsDiffer(a, b map[string]string) string {
	names := make(map[string]bool, len(a)+len(b))
	for name := range a {
		names[name] = true
	}
	for name := range b {
		names[name] = true
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		if a[name] != b[name] {
			return fmt.Sprintf("%s is %q in one and %q in the other", name, a[name], b[name])
		}
	}
	return ""
}
