package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

/*
A field the plugin does not write is a refusal, not a column of zeros.

That was the failure mode of the hand-written 38-of-112 subset: a field renamed
on one side read as zero on the other with nothing said, and the generated
record fixed it once already.
*/
func TestUnknownFieldIsRefused(t *testing.T) {
	_, err := selectField(writeLines(t, ""), "robot_kills_but_spelt_wrong", nil)
	if err == nil {
		t.Fatal("an invented field name was accepted")
	}
	if !strings.Contains(err.Error(), "robot_kills") {
		t.Fatalf("the refusal does not name the fields that exist: %v", err)
	}
}

func TestFieldPicksItsOwnLines(t *testing.T) {
	path := writeLines(t,
		`{"event":"wave_end","wave":1,"result":"cleared","robot_kills":40}`,
		`{"event":"wave_end","wave":2,"result":"lost","robot_kills":10}`,
		`{"event":"wave_end","wave":3,"result":"cleared","robot_kills":30}`,
		`{"event":"bot","wave":1,"nearest_enemy":800}`,
	)

	got, err := selectField(path, "robot_kills", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 3 {
		t.Fatalf("read %d values, want the three wave lines", got.Count)
	}
	if got.Median != 30 || got.Min != 10 || got.Max != 40 {
		t.Fatalf("min %v median %v max %v, want 10 30 40", got.Min, got.Median, got.Max)
	}
}

func TestWhereNarrowsTheLines(t *testing.T) {
	path := writeLines(t,
		`{"event":"wave_end","wave":1,"result":"cleared","robot_kills":40}`,
		`{"event":"wave_end","wave":2,"result":"lost","robot_kills":10}`,
	)

	got, err := selectField(path, "robot_kills", []string{"result=lost"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 1 || got.Median != 10 {
		t.Fatalf("read %d values with median %v, want the one lost wave", got.Count, got.Median)
	}
}

// A field that is not a number is counted by value, which is how "which action
// was the bot in" is asked without a script.
func TestAStringFieldIsCounted(t *testing.T) {
	path := writeLines(t,
		`{"event":"bot","action":"MvMEngineerIdle"}`,
		`{"event":"bot","action":"MvMEngineerIdle"}`,
		`{"event":"bot","action":""}`,
	)

	got, err := selectField(path, "action", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Values["MvMEngineerIdle"] != 2 || got.Values[""] != 1 {
		t.Fatalf("counted %v, want two idles and one empty stack", got.Values)
	}
}

func TestWhereOnAnUnknownFieldIsRefused(t *testing.T) {
	if _, err := selectField(writeLines(t, ""), "robot_kills", []string{"nonsense=1"}); err == nil {
		t.Fatal("a filter on an invented field was accepted")
	}
}

func writeLines(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "results.jsonl")
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
