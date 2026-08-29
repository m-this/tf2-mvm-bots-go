package spbody

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
)

/*
	The DHook shape

A native or an SDKCall is a call the plugin makes, so the emitted body writes
it where the Go wrote it. A DHook is the other direction: the engine enters the
plugin, with the arguments in a DHookParam and the answer expected in a
DHookReturn. That unpacking is the same every time and is generated beside the
body rather than written into it, so the Go stays a plain function that can be
called from a test.

The convention is one sentence: the first parameter is the hooked object, and a
hook that can answer for the engine returns whether it does, then what it
answers. A hook that returns nothing never supercedes.
*/

const dhookDirective = "//sp:dhook"

func hasDHook(d *ast.FuncDecl) bool {
	if d.Doc == nil {
		return false
	}
	for _, c := range d.Doc.List {
		if strings.TrimSpace(c.Text) == dhookDirective {
			return true
		}
	}
	return false
}

func (e *emitter) dhookWrapper(d *ast.FuncDecl, sig *types.Signature) {
	e.inHook = true
	defer func() { e.inHook = false }()

	params := sig.Params()
	if params.Len() == 0 {
		e.fail(d.Pos(), "a DHook with no parameters; the first one is the hooked object")
		return
	}
	results := sig.Results()
	if results.Len() != 0 && results.Len() != 2 {
		e.fail(d.Pos(), "a DHook returning %d values; return nothing, or whether it supercedes and what it answers", results.Len())
		return
	}
	if results.Len() == 2 {
		if b, ok := results.At(0).Type().Underlying().(*types.Basic); !ok || b.Info()&types.IsBoolean == 0 {
			e.fail(d.Pos(), "a DHook whose first result is %s; it says whether the hook supercedes, so it is a bool", results.At(0).Type())
			return
		}
	}

	name := e.cfg.Prefix + d.Name.Name
	e.line("public MRESReturn DHook_%s(int pThis, DHookReturn hReturn, DHookParam hParams)", name)
	e.line("{")
	e.indent++

	args := []string{"pThis"}
	for i := 1; i < params.Len(); i++ {
		p := params.At(i)
		tag, dims, err := e.spType(p.Type())
		if err != nil || len(dims) > 0 {
			e.fail(d.Pos(), "the DHook parameter %s has no DHookParam form", p.Name())
			return
		}
		local := e.ident(d.Pos(), p.Name())
		e.line("%s = %s;", declare(tag, local, nil), dhookGet(tag, i))
		args = append(args, local)
	}

	if results.Len() == 0 {
		e.line("%s(%s);", name, strings.Join(args, ", "))
		e.line("return MRES_Ignored;")
		e.indent--
		e.line("}")
		e.blank()
		return
	}

	answer := results.At(1)
	tag, dims, err := e.spType(answer.Type())
	if err != nil || len(dims) > 0 {
		e.fail(d.Pos(), "the DHook answer %s has no DHookReturn form", answer.Name())
		return
	}
	local := e.ident(d.Pos(), answer.Name())
	e.line("%s;", declare(tag, local, nil))
	e.line("if (%s(%s))", name, strings.Join(append(args, local), ", "))
	e.line("{")
	e.indent++
	e.line("hReturn.Value = %s;", local)
	e.line("return MRES_Supercede;")
	e.indent--
	e.line("}")
	e.line("return MRES_Ignored;")
	e.indent--
	e.line("}")
	e.blank()
}

// dhookGet reads argument n. DHookParam answers in cells, so a bool and a tagged
// integer are the same read with the tag put back on.
func dhookGet(tag string, n int) string {
	switch tag {
	case "float":
		return fmt.Sprintf("hParams.GetFloat(%d)", n)
	case "int":
		return fmt.Sprintf("hParams.Get(%d)", n)
	default:
		return fmt.Sprintf("view_as<%s>(hParams.Get(%d))", tag, n)
	}
}
