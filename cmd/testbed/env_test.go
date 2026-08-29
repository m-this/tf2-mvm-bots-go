package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// Two runners is not a slow run, it is two runs pulling each other's map out.
// The second one has to be told, not queued.
func TestTheSecondRunnerIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")

	release, err := hold(path)
	if err != nil {
		t.Fatalf("the first runner could not take the lock: %v", err)
	}

	if _, err := hold(path); err == nil {
		t.Fatal("the second runner took the lock as well")
	} else if !strings.Contains(err.Error(), "already in use") {
		t.Errorf("the refusal reads %q, and should say who has it", err)
	}

	release()

	again, err := hold(path)
	if err != nil {
		t.Fatalf("the lock was not released: %v", err)
	}
	again()
}

func TestAnArmIsNameAndCvars(t *testing.T) {
	var list arms
	if err := list.Set("on:a=1,b=2"); err != nil {
		t.Fatal(err)
	}
	if list[0].name != "on" || list[0].cvars != "a=1,b=2" {
		t.Errorf("parsed %+v", list[0])
	}
	if err := list.Set("nocolon"); err == nil {
		t.Error("an arm with no colon was accepted")
	}
}
