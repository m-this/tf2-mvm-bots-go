package threat_test

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/threat"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

const shippedFile = "source/redbots3/nextbot_behavior.sp"

func shippedSource(t *testing.T) string {
	t.Helper()

	upstream.SkipOrFail(t)
	body, err := upstream.Read(shippedFile)
	if err != nil {
		t.Fatalf("reading %s at %s: %v", shippedFile, upstream.Rev, err)
	}
	return body
}

// functionBody cuts one function out by matching braces from the line its
// signature is on.
func functionBody(t *testing.T, source, signature string) string {
	t.Helper()

	start := strings.Index(source, signature)
	if start < 0 {
		t.Fatalf("%s is not in the pinned revision: move the pin deliberately", signature)
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

var (
	defineFloat  = regexp.MustCompile(`(?m)^#define\s+(THREAT_\w+_RANGE)\s+([0-9.]+)`)
	priorityName = regexp.MustCompile(`THREAT_PRIORITY_[A-Z_]+`)
	// Anchored on the return, because THREAT_PRIORITY_RANGE is a distance and
	// not an answer, and the broad match takes it for one.
	priorityReturn = regexp.MustCompile(`return\s+(THREAT_PRIORITY_[A-Z_]+)\s*;`)
	anyCall        = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
)

// TestTheRangesAreTheShippedOnes. Both were set by measurement, and a port that
// quietly rounded one would change the behaviour it was meant to preserve.
func TestTheRangesAreTheShippedOnes(t *testing.T) {
	src := shippedSource(t)

	want := map[string]float32{
		"THREAT_URGENT_RANGE":   threat.UrgentRange,
		"THREAT_PRIORITY_RANGE": threat.PriorityRange,
	}
	found := map[string]bool{}
	for _, m := range defineFloat.FindAllStringSubmatch(src, -1) {
		expected, ours := want[m[1]]
		if !ours {
			continue
		}
		found[m[1]] = true
		v, err := strconv.ParseFloat(m[2], 32)
		if err != nil {
			t.Fatalf("%s is defined as %q: %v", m[1], m[2], err)
		}
		if float32(v) != expected {
			t.Errorf("%s is %g in the plugin and %g here", m[1], v, expected)
		}
	}
	for name := range want {
		if !found[name] {
			t.Errorf("%s is not in the pinned plugin any more", name)
		}
	}
}

/*
	TestTheEnumIsTheShippedOrder

The priorities are compared with greater-than, so their order is the behaviour
and not a listing. A name inserted in the middle upstream re-ranks everything
below it, which is the features.sp bug in another file.
*/
func TestTheEnumIsTheShippedOrder(t *testing.T) {
	src := shippedSource(t)

	block := functionBody(t, src, "\nenum\n{\n\tTHREAT_PRIORITY_NONE")
	shipped := priorityName.FindAllString(block, -1)

	ours := make([]string, 0, len(threat.Priorities()))
	for _, p := range threat.Priorities() {
		ours = append(ours, p.Enum())
	}
	if !slices.Equal(shipped, ours) {
		t.Errorf("the pinned enum is\n\t%v\nand this package declares\n\t%v", shipped, ours)
	}
}

/*
	TestEveryShippedAnswerIsAnOutcome

Both directions. An answer the plugin returns and this package cannot produce is
a branch the port dropped; an outcome this package declares that the shipped
function never returns is one the port invented. Either is a behaviour change,
which mvm-z83.41 does not allow to ride along with a move.
*/
func TestEveryShippedAnswerIsAnOutcome(t *testing.T) {
	body := functionBody(t, shippedSource(t), "static int ThreatPriority(int threat, float rangeSq)")

	var returned []string
	for _, m := range priorityReturn.FindAllStringSubmatch(body, -1) {
		if !slices.Contains(returned, m[1]) {
			returned = append(returned, m[1])
		}
	}
	if len(returned) == 0 {
		t.Fatal("no return of a priority in the pinned body: the function changed shape")
	}

	reachable := map[string]bool{}
	for got := range threat.Threats {
		reachable[threat.PriorityOf(got).Enum()] = true
	}

	for _, name := range returned {
		if !reachable[name] {
			t.Errorf("the shipped function returns %s and nothing here reaches it", name)
		}
	}
	for name := range reachable {
		if !slices.Contains(returned, name) {
			t.Errorf("this package answers %s and the shipped function never returns it", name)
		}
	}
	t.Logf("%d answers in the shipped body, %d reachable here", len(returned), len(reachable))
}

/*
	TestTheRecordCarriesEverythingTheShippedFunctionReads

The one that says the port dropped no input. Every call the shipped body makes
about the threat has to be a field of the record, and the record must have no
field that stands for nothing the shipped body reads.

An input read by the plugin and missing from the record would be a decision this
package makes with less than the plugin had, and it would still pass a
differential test, because the test drives both from the same record.
*/
func TestTheRecordCarriesEverythingTheShippedFunctionReads(t *testing.T) {
	body := functionBody(t, shippedSource(t), "static int ThreatPriority(int threat, float rangeSq)")

	// The one call that is not a question about the threat: the switch reads
	// the class through it, and the class is a field.
	fields := map[string]string{
		"BaseEntity_IsPlayer": "IsPlayer",
		"IsClientInGame":      "InGame",
		"TF2_GetPlayerClass":  "Class",
		"TF2_IsMiniBoss":      "Giant",
		"TF2_HasTheFlag":      "Carrier",
	}

	var read []string
	for _, m := range anyCall.FindAllStringSubmatch(body, -1) {
		name := m[1]
		if name == "switch" || name == "if" || slices.Contains(read, name) {
			continue
		}
		read = append(read, name)
	}

	for _, name := range read {
		if _, ok := fields[name]; !ok {
			t.Errorf("the shipped body calls %s and the record has no field for it", name)
		}
	}
	for name := range fields {
		if !slices.Contains(read, name) {
			t.Errorf("the record carries %s for %s, and the pinned body does not call it", fields[name], name)
		}
	}
	t.Logf("%d engine reads in the shipped body, all of them fields: %v", len(read), read)
}
