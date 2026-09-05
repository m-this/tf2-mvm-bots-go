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
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "nolint") {
			continue
		}
		e.line("%s", strings.TrimRight("// "+line, " "))
	}
}
