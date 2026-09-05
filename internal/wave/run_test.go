package wave_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

func TestReadRunFindsTheRecordAndSaysWhenThereIsNone(t *testing.T) {
	dir := t.TempDir()
	with := filepath.Join(dir, "with.jsonl")
	body := `{"event":"run","tag":"t","arm":"on","machine":{"host":"bed","mem_available_kb":4194304,"load_1m":0.5,"extensions":{"x.so":"abc"}}}` + "\n" +
		`{"event":"wave_end","wave":1,"result":"cleared"}` + "\n"
	if err := os.WriteFile(with, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	run, found, err := wave.ReadRun(with)
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if run.Arm != "on" || run.Machine.Host != "bed" || run.Machine.Extensions["x.so"] != "abc" {
		t.Errorf("run %+v", run)
	}

	without := filepath.Join(dir, "without.jsonl")
	if err := os.WriteFile(without, []byte(`{"event":"wave_end","wave":1}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := wave.ReadRun(without); found || err != nil {
		t.Errorf("a file with no record: found=%v err=%v", found, err)
	}
}
