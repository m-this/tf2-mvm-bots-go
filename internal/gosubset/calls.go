package gosubset

import (
	"fmt"
	"go/ast"
)

func (c *checker) checkCall(e *ast.CallExpr) {
	if e.Ellipsis.IsValid() {
		c.refuse(e.Ellipsis, "a call spreading an argument with ...",
			"pass the array and a count; the subset has no variadic calls")
	}
	c.checkCallee(e.Fun)
	for _, arg := range e.Args {
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
	c.refuse(sel.Pos(), "a method call",
		"call a plain function with the value as its first parameter; the body generator emits no methodmaps")
}
