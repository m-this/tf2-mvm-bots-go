package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRun lays down one attempt's file: a run record and two wave ends.
func writeRun(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A run read back off its files is the run, arms and all.
func TestRereadRebuildsTheArms(t *testing.T) {
	dir := t.TempDir()
	for _, arm := range []string{"on", "off"} {
		for round := 1; round <= 2; round++ {
			writeRun(t, dir, "t-"+arm+"-"+string(rune('0'+round))+".jsonl", strings.Join([]string{
				`{"event":"run","tag":"t","arm":"` + arm + `","map":"mvm_decoy","machine":{"host":"lab","load_1m":1.0,"mem_available_kb":9999999}}`,
				`{"event":"wave_begin","wave":1}`,
				`{"event":"wave_end","wave":1,"result":"cleared","robot_kills":60}`,
				`{"event":"wave_begin","wave":2}`,
				`{"event":"wave_end","wave":2,"result":"cleared","robot_kills":130}`,
			}, "\n"))
		}
	}

	got, err := rereadArms(dir, "t")
	if err != nil {
		t.Fatal(err)
	}
	arms := got.Arms
	if got.Map != "mvm_decoy" {
		t.Errorf("the map read as %q", got.Map)
	}
	if got.Older != 0 {
		t.Errorf("%d files read as older than the run record, want 0", got.Older)
	}
	if len(arms) != 2 {
		t.Fatalf("read %d arms, want 2", len(arms))
	}
	// The control goes last, which is the order the comparison reads.
	if arms[len(arms)-1].Name != "off" {
		t.Errorf("the control is %q, want off", arms[len(arms)-1].Name)
	}
	for _, a := range arms {
		if a.Attempts != 2 || len(a.Results) != 4 || len(a.Machines) != 2 {
			t.Errorf("%s read as %d attempts, %d waves, %d machines", a.Name, a.Attempts, len(a.Results), len(a.Machines))
		}
	}
}

// A file written before the run record still has its arm in its name, and
// refusing the whole read over it would make this useless on the archive it
// exists for.
func TestRereadTakesAFileWithNoRunRecord(t *testing.T) {
	dir := t.TempDir()
	writeRun(t, dir, "old-on-1.jsonl", strings.Join([]string{
		`{"event":"wave_begin","wave":1}`,
		`{"event":"wave_end","wave":1,"map":"mvm_decoy","result":"lost","robot_kills":40}`,
	}, "\n"))

	got, err := rereadArms(dir, "old")
	if err != nil {
		t.Fatal(err)
	}
	if got.Older != 1 {
		t.Errorf("%d files read as older, want 1", got.Older)
	}
	if got.Map != "mvm_decoy" {
		t.Errorf("the map did not come off the waves: %q", got.Map)
	}
	if len(got.Arms) != 1 || got.Arms[0].Name != "on" || len(got.Arms[0].Machines) != 0 {
		t.Errorf("the arm read as %+v", got.Arms)
	}
}

func TestRereadSaysWhenThereIsNothingToRead(t *testing.T) {
	if _, err := rereadArms(t.TempDir(), "nothing"); err == nil {
		t.Error("an empty directory produced a run")
	}
}
