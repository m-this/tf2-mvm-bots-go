package spbody

import (
	"go/ast"
	"strings"
)

/*
The comments come across with the code

A reviewer reads the generated SourcePawn beside what it does, and a decision
with its reason removed reads as arbitrary. The doc comment of a function and
the comment above a statement are emitted where they were written. What is
left out is what speaks to a tool and not a reader: the sp: directives, nolint
and the rest of the //word: family, which CommentGroup.Text already drops.
*/

// leading writes the comments that stand above a node.
func (e *emitter) leading(node ast.Node) {
	for _, group := range e.comments[node] {
		if group.End() < node.Pos() {
			e.comment(group)
		}
	}
}

// trailing writes the comments that follow a node on its own line.
func (e *emitter) trailing(node ast.Node) {
	for _, group := range e.comments[node] {
		if group.Pos() > node.End() {
			e.comment(group)
		}
	}
}

// comment writes one group as line comments.
func (e *emitter) comment(group *ast.CommentGroup) {
	text := strings.TrimRight(group.Text(), "\n")
	if text == "" {
		return
	}
	lines := undent(strings.Split(text, "\n"))

	// The opening marker leaves a space in front of the first line, and it is
	// never one somebody typed for effect.
	lines[0] = strings.TrimPrefix(lines[0], " ")

	for _, line := range lines {
		if strings.HasPrefix(line, "nolint") {
			continue
		}
		e.line("%s", strings.TrimRight("// "+line, " "))
	}
}

// undent takes the block's own indentation off, and leaves the shape inside it.
//
// A block comment written inside a for loop carries that loop's tabs on every
// line after the first, and Text() keeps them: the emitted SourcePawn came out
// with the comment's second line indented past its own opening. The longest run
// of leading whitespace every non-empty line shares is the block's, so removing
// that leaves a list or an example indented as it was written.
func undent(lines []string) []string {
	if len(lines) < 2 {
		return lines
	}

	// The first line follows the opening marker on its own line and carries no
	// indentation of its own, so it says nothing about the block's.
	prefix, found := "", false

	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		lead := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		if !found {
			prefix, found = lead, true

			continue
		}

		for !strings.HasPrefix(lead, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}

	if prefix == "" {
		return lines
	}

	out := make([]string, 0, len(lines))
	out = append(out, lines[0])

	for _, line := range lines[1:] {
		out = append(out, strings.TrimPrefix(line, prefix))
	}

	return out
}
