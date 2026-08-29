package gosubset

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (c *checker) checkCall(e *ast.CallExpr) {
	if e.Ellipsis.IsValid() {
		c.refuse(e.Ellipsis, "a call spreading an argument with ...",
			"pass the array and a count; the subset has no variadic calls")
	}
	c.checkCallee(e.Fun)
	for _, arg := range e.Args {
		// A string literal is allowed here and nowhere else. SourcePawn
		// needs a real string for an entity classname, and there is no
		// int32 identifier that would do instead; everything else the
		// subset would use one for stays refused.
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			continue
		}
		c.checkExpr(arg)
	}
}

func (c *checker) checkCallee(fun ast.Expr) {
	switch f := fun.(type) {
	case *ast.Ident:
		c.checkCalleeName(f)
	case *ast.ParenExpr:
		c.checkCallee(f.X)
	case *ast.SelectorExpr:
		c.checkCalleeSelector(f)
	case *ast.ArrayType, *ast.StructType, *ast.MapType, *ast.ChanType, *ast.InterfaceType, *ast.FuncType, *ast.StarExpr:
		c.checkType(fun) // a conversion to a type the subset may not have
	default:
		c.checkExpr(fun)
	}
}

func (c *checker) checkCalleeName(id *ast.Ident) {
	name := id.Name
	if allowedBuiltins[name] || c.funcs[name] || c.natives[name] || c.types[name] || basicTypes[name] {
		return
	}
	if fix, refused := refusedBuiltins[name]; refused {
		c.refuse(id.Pos(), fmt.Sprintf("a call to the builtin %s", name), fix)
		return
	}
	if fix, refused := refusedTypes[name]; refused {
		c.refuse(id.Pos(), fmt.Sprintf("a conversion to %s", name), fix)
		return
	}
	c.refuse(id.Pos(), fmt.Sprintf("a call to the unknown function %s", name),
		"the subset calls functions declared in this package and generated native bindings; declare it, or add the binding")
}

func (c *checker) checkCalleeSelector(sel *ast.SelectorExpr) {
	if id, ok := sel.X.(*ast.Ident); ok {
		if _, isImport := c.imports[id.Name]; isImport {
			c.checkSelector(sel)
			return
		}
	}
	/* A method call on something that is not an import.

	Accepted here and judged by the emitter, which has the types. SourceMod's
	API is methodmaps: myBot.GetVisionInterface() has no plain function
	behind it, so refusing every method call refused the engine. What the
	emitter accepts is a method the extern package declares on a type
	carrying an //sp:tag, and it refuses the rest by name.

	This is the one rule this checker cannot decide, because deciding it
	needs to know what the receiver is. */
	if id, ok := sel.X.(*ast.Ident); ok && c.packageNames[id.Name] {
		// A package this configuration maps, used by a file that did not
		// import it. An import is file scoped, and a selector that reads
		// as a package in one file and a variable in the next is the
		// kind of thing nobody notices until it compiles differently.
		c.refuse(sel.Pos(), fmt.Sprintf("%s.%s, in a file that does not import %s", id.Name, sel.Sel.Name, id.Name),
			"import it in this file, or call something this one can see")
		return
	}
	c.checkExpr(sel.X)
}
