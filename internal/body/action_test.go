package body_test

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/spaction"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	TestGeneratedActionMatchesTheShippedOne

mvm-z83.11 asks for one action generated beside the hand written one and diffed,
and this is the diff. It is not the differential test the bodies get: an action
callback is entered by the engine with a path, a nextbot and a game clock behind
it, and none of that exists under spshell.

So what is compared is two things, and it is worth being precise about which:

  - the declaration line, byte for byte. That is the engine's, not ours: a
    callback declared with the wrong parameters is entered with the arguments in
    the wrong places and compiles perfectly.
  - the sequence of functions the body calls, in order, by name and not by
    receiver. That catches a dropped call, an extra one and a reordering, which
    is what a bad translation looks like. The receiver is left out because a
    local's name is the port's to choose, and calling a method on the wrong
    methodmap does not compile.

It does not compare the body text. The generator writes braces around a single
statement where the plugin does not, and folds 2.0 * sapRange to 80.0, and
neither is a difference in what runs. The body-level proof is the differential
test the pure bodies get, and an action cannot have one.
*/
func TestGeneratedActionMatchesTheShippedOne(t *testing.T) {
	// No skip: the shipped text comes from the snapshot under
	// internal/upstream. See TestGeneratedBodiesMatchTheShippedOnes.

	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating: %v", err)
	}
	for _, action := range body.Actions {
		t.Run(action.Dir, func(t *testing.T) {
			compareAction(t, action, string(generated[action.Out]))
		})
	}
}

func compareAction(t *testing.T, action body.Body, got string) {
	t.Helper()

	// At the pin, not in the working tree. The working tree no longer has
	// the hand written file: the port took it, which is the whole point, and
	// a comparison that read the tree would have quietly stopped comparing
	// anything the moment it succeeded.
	shipped, err := upstream.ReadAt(action.Rev, strings.Split(action.Shipped, "/")...)
	if err != nil {
		t.Fatalf("reading the shipped behaviour at %s: %v", upstream.Rev, err)
	}

	declared, err := spaction.Read(filepath.Join("../..", action.Dir))
	if err != nil {
		t.Fatalf("reading the action: %v", err)
	}

	for _, name := range declared.Has {
		t.Run(name, func(t *testing.T) {
			want, ok := callbackOf(shipped, declared.Prefix+"_"+name)
			if !ok {
				t.Fatalf("the shipped file has no %s_%s", declared.Prefix, name)
			}
			have, ok := callbackOf(got, declared.Prefix+"_"+name)
			if !ok {
				t.Fatalf("the generated file has no %s_%s", declared.Prefix, name)
			}
			if wantDecl, haveDecl := declOf(want), declOf(have); wantDecl != haveDecl {
				t.Errorf("the declaration is not the engine's:\nshipped:   %s\ngenerated: %s", wantDecl, haveDecl)
			}
			if wantCalls, haveCalls := callsIn(want), callsIn(have); !slices.Equal(wantCalls, haveCalls) {
				t.Errorf("the body calls a different sequence:\nshipped:   %v\ngenerated: %v", wantCalls, haveCalls)
			}
		})
	}

	// The constructor is the one place a missing callback is invisible: the
	// action just stops doing something and nothing says so.
	for _, wire := range regexp.MustCompile(`action\.(\w+)\s*=`).FindAllStringSubmatch(shipped, -1) {
		if !strings.Contains(got, "action."+wire[1]+" =") {
			t.Errorf("the shipped constructor wires %s and the generated one does not", wire[1])
		}
	}
}

// callbackOf pulls one function out of a file and normalises it the way the
// wave writer comparison does: comments go, runs of whitespace become one space.
func callbackOf(src, name string) (string, bool) {
	i := strings.Index(src, name+"(")
	if i < 0 {
		return "", false
	}
	// Back up to the start of the declaration line.
	if start := strings.LastIndexByte(src[:i], '\n'); start >= 0 {
		i = start + 1
	}
	j := strings.Index(src[i:], "\n}\n")
	if j < 0 {
		return "", false
	}
	body := src[i : i+j+3]
	body = reBlock.ReplaceAllString(body, "")
	body = reLine.ReplaceAllString(body, "")
	return strings.TrimSpace(reRun.ReplaceAllString(body, " ")), true
}

// declOf is the declaration line, which is the part the engine decides.
func declOf(fn string) string {
	decl, _, _ := strings.Cut(fn, "{")
	return strings.TrimSpace(decl)
}

/*
	callsIn is every function the body calls, in order

Names only. The arguments are where the folding and the brace style show up, and
neither changes what runs; a call that went missing or moved does.
*/
func callsIn(fn string) []string {
	_, body, _ := strings.Cut(fn, "{")
	found := reCall.FindAllStringSubmatch(body, -1)
	out := make([]string, 0, len(found))
	for _, m := range found {
		// if and switch are not calls. Counting them made the comparison
		// sensitive to whether a result was tested inline or assigned
		// first, which Go often cannot choose: a function with two
		// results has to be assigned before it can be tested.
		if keywords[m[1]] {
			continue
		}
		// The name, not the receiver. A local's name is the port's to
		// choose and Go renames potential_victims to potentialVictims;
		// the receiver is checked by spcomp anyway, since calling a
		// method on the wrong methodmap does not compile.
		name := m[1]
		if i := strings.LastIndexByte(name, '.'); i >= 0 {
			name = name[i+1:]
		}
		out = append(out, name)
	}
	return out
}

// keywords read as calls to the pattern above and are not.
/*
	keywords are the things that read as a call and are not one

Control flow, and the three conversions. float() and view_as() change how a value
is read and nothing else: the port writes them where Go needs a conversion and
the plugin leaves the coercion implicit, which is a difference in spelling rather
than in what runs.
*/
var keywords = map[string]bool{
	"if": true, "for": true, "switch": true, "while": true, "return": true,
	"sizeof": true, "view_as": true, "float": true,
}

var (
	reCall  = regexp.MustCompile(`([A-Za-z_][\w.]*)\s*\(`)
	reBlock = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reLine  = regexp.MustCompile(`//[^\n]*`)
	reRun   = regexp.MustCompile(`\s+`)
)
