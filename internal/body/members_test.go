package body_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

/*
	blockHeader is the line an enum struct or a methodmap opens with

Both keep their methods inside their braces, so a method is not something the
top-level comparison can see: emittedNames is anchored at the start of a line
and every method is indented. This is the second pass that does see them.
*/
var blockHeader = regexp.MustCompile(`(?m)^(?:enum struct|methodmap) (\w+)`)

// memberHeader is a method declared inside one of those blocks.
var memberHeader = regexp.MustCompile(`(?m)^\t(?:public )?[A-Za-z_]\w*(?:\[\])? (\w+)\(`)

/*
	compareMembers checks the methods inside every generated type

A type the port carries over keeps the plugin's name, so the block to compare
against is the one with that name in the shipped file. Within it a method is
matched by its own name, and the same two things are compared as for a plain
function: the shape of the declaration and the sequence of calls.

Without this the three enum structs and the upgrade methodmaps would generate
and prove nothing, which is the same hole the top-level comparison already
refuses for a package with no functions.
*/
func compareMembers(t *testing.T, got, shipped string) int {
	t.Helper()

	compared := 0

	for _, block := range blockHeader.FindAllStringSubmatchIndex(got, -1) {
		name := got[block[2]:block[3]]

		body, ok := blockBody(got, block[0])
		if !ok {
			t.Fatalf("the generated file opens %s and does not close it", name)
		}

		want, ok := shippedBlock(shipped, name)
		if !ok {
			// A record this port added, or one from another file:
			// neither is a difference in what was replaced.
			continue
		}

		for _, m := range memberHeader.FindAllStringSubmatch(body, -1) {
			member := m[1]

			if reshaped[name+"."+member] != "" {
				continue
			}

			shippedMember, ok := memberOf(want, member)
			if !ok {
				continue
			}
			generatedMember, ok := memberOf(body, member)
			if !ok {
				t.Fatalf("%s declares %s and then does not hold it", name, member)
			}

			compared++

			t.Run(name+"."+member, func(t *testing.T) {
				if a, b := shape(shippedMember), shape(generatedMember); a != b {
					t.Errorf("the declaration differs:\nshipped:   %s\ngenerated: %s", a, b)
				}
				wantFrees, haveFrees := frees(shippedMember), frees(generatedMember)
				if wantFrees != haveFrees {
					t.Errorf("the body frees %d handles and the shipped one frees %d", haveFrees, wantFrees)
				}

				a := withoutCloses(callsIn(shippedMember))
				b := withoutCloses(callsIn(generatedMember))
				if !slices.Equal(a, b) {
					t.Errorf("the body calls a different sequence:\nshipped:   %v\ngenerated: %v", a, b)
				}
			})
		}
	}

	return compared
}

// blockBody is the braced block that starts at the header given.
func blockBody(src string, at int) (string, bool) {
	open := strings.Index(src[at:], "\n{\n")
	if open < 0 {
		return "", false
	}
	end := strings.Index(src[at+open:], "\n}\n")
	if end < 0 {
		return "", false
	}
	return src[at+open : at+open+end], true
}

// shippedBlock is the block of that name in the shipped file.
func shippedBlock(shipped, name string) (string, bool) {
	at := regexp.MustCompile(`(?m)^(?:enum struct|methodmap) ` + regexp.QuoteMeta(name) + `\b`)

	loc := at.FindStringIndex(shipped)
	if loc == nil {
		return "", false
	}
	return blockBody(shipped, loc[0])
}

/*
	memberOf is one method of a block, declaration and body

A method's closing brace is one tab in, which is what tells it apart from the
block's own. Comments are dropped and runs of space collapsed, the same way
callbackOf does it for a plain function.
*/
func memberOf(block, name string) (string, bool) {
	at := regexp.MustCompile(`(?m)^\t(?:public )?[A-Za-z_]\w*(?:\[\])? ` + regexp.QuoteMeta(name) + `\(`)

	loc := at.FindStringIndex(block)
	if loc == nil {
		return "", false
	}

	rest := block[loc[0]:] + "\n"

	end := strings.Index(rest, "\n\t}\n")
	if end < 0 {
		return "", false
	}

	body := rest[:end+4]
	body = reBlock.ReplaceAllString(body, "")
	body = reLine.ReplaceAllString(body, "")

	return strings.TrimSpace(reRun.ReplaceAllString(body, " ")), true
}
