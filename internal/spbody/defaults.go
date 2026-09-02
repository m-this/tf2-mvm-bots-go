package spbody

import (
	"fmt"
	"go/ast"
	"strings"
)

/*
	Default parameter values

Go has none and SourcePawn does, and the plugin's call sites rely on them: nine
of util.sp's scan functions are called with three arguments and declared with
six. A port that dropped the defaults would compile nowhere until every caller
moved in the same commit, which is the flag day this epic is trying not to have.

So a body may declare them, on the Go function, as

	//sp:default bGiantsOnly false

The value is written into the SourcePawn verbatim, so it is spelled the way
SourcePawn spells it: TFClass_Unknown, not the Go name for the same constant.

The Go side is unaffected: a Go caller passes every argument, so nothing about
the behaviour depends on a default. It is a compatibility shim for the
SourcePawn that has not been ported yet, and it goes when the callers do.
*/

const defaultDirective = "//sp:default"

// defaultsOf reads the declared defaults, by parameter name.
func (e *emitter) defaultsOf(d *ast.FuncDecl) map[string]string {
	if d.Doc == nil {
		return nil
	}
	out := make(map[string]string)
	/* Line by line, because a block comment is one entry

	A block doc comment arrives as a single ast.Comment whose text is the
	whole block, so testing the entry for the prefix found nothing and the
	directive was dropped without a word: RemoveAllDefenderBots lost both its
	defaults, and only the shipped comparison noticed. Every other directive
	reader already walks the lines. */
	for _, c := range d.Doc.List {
		for line := range strings.Lines(c.Text) {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, defaultDirective) {
				continue
			}
			rest := strings.TrimSpace(strings.TrimPrefix(line, defaultDirective))
			name, value, ok := strings.Cut(rest, " ")
			if !ok || strings.TrimSpace(value) == "" {
				e.fail(d.Pos(), "the directive %q needs a parameter name and a value", line)
				continue
			}
			if _, dup := out[name]; dup {
				e.fail(d.Pos(), "%s is given a default twice", name)
				continue
			}
			out[name] = strings.TrimSpace(value)
		}
	}
	return out
}

// applyDefaults appends the declared default to each parameter that has one,
// and refuses the shape SourcePawn will not take: a parameter with a default
// followed by one without, which no caller could ever omit.
func (e *emitter) applyDefaults(d *ast.FuncDecl, names []string, params []string, defaults map[string]string) []string {
	if len(defaults) == 0 {
		return params
	}
	seen := 0
	given := false
	for i, name := range names {
		value, ok := defaults[name]
		if !ok {
			if given {
				e.fail(d.Pos(), "%s has no default and follows one that does; SourcePawn defaults every parameter after the first defaulted one", name)
			}
			continue
		}
		given = true
		seen++
		params[i] = fmt.Sprintf("%s = %s", params[i], value)
	}
	if seen != len(defaults) {
		for name := range defaults {
			if !contains(names, name) {
				e.fail(d.Pos(), "%s is given a default and is not a parameter", name)
			}
		}
	}
	return params
}

func contains(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
