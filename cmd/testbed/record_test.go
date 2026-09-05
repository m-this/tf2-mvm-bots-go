package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/wave"
)

func TestTheRunRecordLeadsTheFileAndCountsForNoWave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.jsonl")
	body := `{"event":"wave_begin","wave":1}` + "\n" + `{"event":"wave_end","wave":1,"outcome":"cleared"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeRunRecord(path, runRecord{Tag: "t", Arm: "on", Plugin: "2.46.0"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	first, _, _ := strings.Cut(string(got), "\n")
	if !strings.HasPrefix(first, `{"event":"run","tag":"t","arm":"on"`) {
		t.Fatalf("first line is %q", first)
	}
	if !strings.HasSuffix(string(got), body) {
		t.Fatal("the server's lines did not follow the record")
	}
	results, err := wave.Read(path)
	if err != nil || len(results) != 1 {
		t.Fatalf("wave.Read gave %d results, %v", len(results), err)
	}
}
