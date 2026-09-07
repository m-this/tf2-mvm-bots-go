package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateRoundTrips(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	want := State{
		Bed: "bed-under-test", Port: "27099", Container: "bed-under-test-srcds-1",
		PID: os.Getpid(), Tag: "x", Map: "mvm_decoy", Arm: "on", Round: 2,
		Attempts: 3, WavesSeen: 1, WavesWanted: 2, Samples: 400,
	}
	if err := writeState(want); err != nil {
		t.Fatal(err)
	}

	got, found, err := readState(want.Bed)
	if err != nil || !found {
		t.Fatalf("readState: %v, found %v", err, found)
	}
	if got.Tag != want.Tag || got.Round != want.Round || got.Samples != want.Samples {
		t.Fatalf("read back %+v, want %+v", got, want)
	}
	if got.UpdatedAt == "" {
		t.Fatal("no updated_at: a reader cannot tell a fresh record from an abandoned one")
	}
	if _, err := time.Parse(time.RFC3339, got.UpdatedAt); err != nil {
		t.Fatalf("updated_at %q is not RFC3339, so a reader cannot compare it with a container log line: %v", got.UpdatedAt, err)
	}
}

func TestNoRecordIsNotAnError(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	_, found, err := readState("a-bed-nobody-has-run")
	if err != nil || found {
		t.Fatalf("readState on a bed with no record: %v, found %v", err, found)
	}
}

/*
A record whose runner is gone reads as ended.

The whole point of the file is a second shell that has no terminal to look at,
and one that only trusted Outcome would wait for a killed runner forever.
*/
func TestRunnerGoneReadsAsEnded(t *testing.T) {
	// Pid 1 exists and is not us; a pid that cannot exist is the dead case.
	dead := State{Bed: "b", PID: 1 << 30}
	if !dead.Ended() || !dead.Stale() {
		t.Fatal("a record whose pid is gone and which wrote no verdict is stale and ended")
	}

	live := State{Bed: "b", PID: os.Getpid()}
	if live.Ended() || live.Stale() {
		t.Fatal("a record this process is writing is neither ended nor stale")
	}

	said := State{Bed: "b", PID: 1 << 30, Outcome: outcomeFinished}
	if !said.Ended() || said.Stale() {
		t.Fatal("a record that says finished is ended and not stale, whatever its pid")
	}
}

// The record is replaced whole, so a reader polling it never gets half of one.
func TestStateIsReplacedAtomically(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)

	for i := range 5 {
		if err := writeState(State{Bed: "b", Round: i}); err != nil {
			t.Fatal(err)
		}
	}
	left, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 {
		t.Fatalf("%d files left behind, want the record alone: %v", len(left), left)
	}
}
