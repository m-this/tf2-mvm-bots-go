package body_test

import (
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
)

// TestNothingIsOwnedTwice is the rule the epic rests on, checked rather than
// remembered. A plugin extern names SourcePawn this repository has not written
// yet; the day it writes it, both would compile and nothing else would notice.
func TestNothingIsOwnedTwice(t *testing.T) {
	if _, err := body.Generate("../.."); err != nil {
		t.Fatalf("generating the bodies: %v", err)
	}
	// The check lives inside Generate, so this proves it fires rather than
	// that it exists: a body named after a plugin extern has to be refused.
	if _, err := body.GenerateWith("../..", "GetAbsOrigin"); err == nil {
		t.Fatal("a body generating GetAbsOrigin was accepted beside the extern that names it")
	} else if !strings.Contains(err.Error(), "delete the extern") {
		t.Errorf("the refusal says %q, and does not say what to do", err)
	}
}
