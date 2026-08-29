package body_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/spbody"
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
	// The name is whichever plugin extern is still declared, because the
	// point of the port is that the list shrinks.
	declared, err := spbody.ExternsFromDir(filepath.Join("../..", body.ExternDir))
	if err != nil {
		t.Fatalf("reading the extern declarations: %v", err)
	}
	name := ""
	for _, x := range declared.Funcs {
		if x.Plugin && (name == "" || x.Func < name) {
			name = x.Func
		}
	}
	if name == "" {
		t.Skip("no plugin externs left: the port owns everything the bodies call")
	}
	if _, err := body.GenerateWith("../..", name); err == nil {
		t.Fatalf("a body generating %s was accepted beside the extern that names it", name)
	} else if !strings.Contains(err.Error(), "delete the extern") {
		t.Errorf("the refusal says %q, and does not say what to do", err)
	}
}
