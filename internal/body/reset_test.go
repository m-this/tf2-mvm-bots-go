package body_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
)

/*
TestANewPerClientArrayHasToBeCleared is the check the shipped ResetNextBot never
had.

The failure it stops is the one that list always had: a behaviour declares a
per-client array, forgets to clear it, and the next bot in the seat starts a
wave believing something about the last one. It was silent before the port,
because the list was flat and long; it was silent after, because each package
clears its own and nothing said whether it did.

The proof is to add one and watch it be refused, which is what this does.
*/
func TestANewPerClientArrayHasToBeCleared(t *testing.T) {
	root := treeCopy(t)
	path := filepath.Join(root, "internal", "action", "campbomb", "carried_test_fixture.go")
	const source = `package campbomb

import "github.com/m-this/tf2-mvm-bots-go/internal/body/slots"

var carriedOver [slots.Count]float32

// Read so the declaration is not dead, and never cleared, which is the point.
func CarriedOver(client int32) float32 { return carriedOver[client] }
`
	// 0o600: the fixture lives in a tree only this test reads.
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	_, err := body.Generate(root)
	if err == nil {
		t.Fatal("a per-client array nothing clears was accepted")
	}
	if !strings.Contains(err.Error(), "carriedOver") {
		t.Errorf("the refusal does not name the array: %v", err)
	}
	if !strings.Contains(err.Error(), "//sp:keep") {
		t.Errorf("the refusal does not say what to do instead: %v", err)
	}
}

// TestAReasonIsEnough is the other half: a package that says why an array
// survives a seat is accepted, and the reason is in the source where a reader
// will find it rather than in a list somewhere else.
func TestAReasonIsEnough(t *testing.T) {
	root := treeCopy(t)
	path := filepath.Join(root, "internal", "action", "campbomb", "kept_test_fixture.go")
	const source = `package campbomb

import "github.com/m-this/tf2-mvm-bots-go/internal/body/slots"

//sp:keep the map decides this, not the bot, so a new bot in the seat keeps it
var mapWide [slots.Count]float32

func MapWide(client int32) float32 { return mapWide[client] }
`
	// 0o600: the fixture lives in a tree only this test reads.
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}

	if _, err := body.Generate(root); err != nil {
		t.Fatalf("an array with a written reason was refused: %v", err)
	}
}
