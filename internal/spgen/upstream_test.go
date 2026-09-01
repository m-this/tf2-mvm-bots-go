package spgen_test

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/spgen"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

// readUpstream is the pinned plugin file this proof reads. The pin, and why
// there is one, is internal/plugin. No plugin tree is a skip; a tree
// repository missing the file at the pin is a failure.
func readUpstream(t *testing.T, path string) string {
	t.Helper()

	body, err := upstream.Read(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return body
}

var suspendFor = regexp.MustCompile(`SuspendFor\((\w+)\(\),\s*"((?:[^"\\]|\\.)*)"\)`)

// functionBody cuts one function out of a SourcePawn file by matching braces
// from the line the signature is on.
func functionBody(t *testing.T, source, signature string) string {
	t.Helper()

	start := strings.Index(source, signature)
	if start < 0 {
		t.Fatalf("%s is not in the pinned revision any more: move the pin deliberately", signature)
	}
	depth, open := 0, -1
	for i := start; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
			if open < 0 {
				open = i
			}
		case '}':
			depth--
			if depth == 0 && open >= 0 {
				return source[open : i+1]
			}
		}
	}
	t.Fatalf("%s has no matching closing brace", signature)
	return ""
}

// TestEveryShippedCallSiteIsInTheEdge is the anti-drift proof. Every
// SuspendFor in GetDesiredBotAction, behaviour and reason string together,
// has to appear exactly once in the edge table, and the edge table must hold
// nothing that is not in the function. That is what stops a reason string
// being quietly renamed on one side: the reason reaches the debug output and
// the test-bed's telemetry, so it is part of the behaviour.
func TestEveryShippedCallSiteIsInTheEdge(t *testing.T) {
	body := functionBody(t, readUpstream(t, "source/redbots3/nextbot_behavior.sp"),
		"Action GetDesiredBotAction(int client, BehaviorAction action)")

	shipped := map[string]int{}
	for _, m := range suspendFor.FindAllStringSubmatch(body, -1) {
		shipped[m[1]+" / "+m[2]]++
	}

	edge := map[string]int{}
	for _, o := range spgen.ActionSelOutcomes {
		if o.Behaviour == "" {
			continue
		}
		edge[o.Behaviour+" / "+o.Reason]++
	}

	for pair, n := range edge {
		if n != 1 {
			t.Errorf("the edge lists %q %d times; one outcome is one call site", pair, n)
		}
		if _, ok := shipped[pair]; !ok {
			t.Errorf("the edge lists %q, which GetDesiredBotAction does not call", pair)
		}
	}
	for pair := range shipped {
		if _, ok := edge[pair]; !ok {
			t.Errorf("GetDesiredBotAction calls %q, and no outcome carries it", pair)
		}
	}

	t.Logf("%d distinct call sites over %d SuspendFor calls", len(shipped), countAll(shipped))
	if t.Failed() {
		t.Log("the pinned function calls:\n" + strings.Join(sortedKeys(shipped), "\n"))
	}
}

func countAll(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("  %s (x%d)", k, v))
	}
	slices.Sort(out)
	return out
}
