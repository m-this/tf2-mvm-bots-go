package body_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
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
  - the sequence of functions the body calls, in order. That catches a dropped
    call, an extra one and a reordering, which is what a bad translation looks
    like.

It does not compare the body text. The generator writes braces around a single
statement where the plugin does not, and folds 2.0 * sapRange to 80.0, and
neither is a difference in what runs. The body-level proof is the differential
test the pure bodies get, and an action cannot have one.
*/
func TestGeneratedActionMatchesTheShippedOne(t *testing.T) {
	root := upstream.SkipOrFail(t)

	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating: %v", err)
	}
	got := string(generated["sourcepawn/spysap.sp"])
	shipped := readUpstreamFile(t, root, "source", "redbots3", "behavior", "spysap.sp")

	for _, name := range []string{"OnStart", "Update", "OnEnd", "OnSuspend", "OnResume"} {
		t.Run(name, func(t *testing.T) {
			want, ok := callbackOf(shipped, "CTFBotSpySap_"+name)
			if !ok {
				t.Fatalf("the shipped file has no CTFBotSpySap_%s", name)
			}
			have, ok := callbackOf(got, "CTFBotSpySap_"+name)
			if !ok {
				t.Fatalf("the generated file has no CTFBotSpySap_%s", name)
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
		out = append(out, m[1])
	}
	return out
}

var (
	reCall  = regexp.MustCompile(`([A-Za-z_][\w.]*)\s*\(`)
	reBlock = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reLine  = regexp.MustCompile(`//[^\n]*`)
	reRun   = regexp.MustCompile(`\s+`)
)
