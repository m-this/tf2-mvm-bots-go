package gosubset

import (
	"fmt"
	"go/ast"
	"go/token"
)

// allowedBuiltins are the builtin calls with a SourcePawn equivalent. len over
// a fixed-length array is a constant, min and max are two lines each.
var allowedBuiltins = map[string]bool{"len": true, "min": true, "max": true}

// refusedBuiltins carries the fix for the builtins a reader reaches for first.
var refusedBuiltins = map[string]string{
	"make":    "declare a fixed-length array; the subset allocates nothing",
	"new":     "declare the value; the subset allocates nothing",
	"append":  "size the array by a constant and carry a count of how much is filled",
	"copy":    "assign the array, or write the loop; SourcePawn copies arrays by assignment",
	"delete":  "the subset has no maps",
	"panic":   "return a bool or a sentinel int32 and let the hand-written SourcePawn decide",
	"recover": "the subset has no panics to recover from",
	"print":   "a generated body prints nothing; return the value and log it at the call site",
	"println": "a generated body prints nothing; return the value and log it at the call site",
	"complex": "SourcePawn has no complex numbers",
	"real":    "SourcePawn has no complex numbers",
	"imag":    "SourcePawn has no complex numbers",
	"clear":   "assign the zero value field by field, or loop over the array",
	"cap":     "the subset has no slices; use len over the fixed-length array",
}

func (c *checker) checkExpr(expr ast.Expr) {
	switch e := expr.(type) {
	case nil:
	case *ast.Ident:
	case *ast.BasicLit:
		c.checkBasicLit(e)
	case *ast.ParenExpr:
		c.checkExpr(e.X)
	case *ast.SelectorExpr:
		c.checkSelector(e)
	case *ast.IndexExpr:
		c.checkExpr(e.X)
		c.checkExpr(e.Index)
	case *ast.BinaryExpr:
		c.checkBinary(e)
	case *ast.UnaryExpr:
		c.checkUnary(e)
	case *ast.CallExpr:
		c.checkCall(e)
	case *ast.CompositeLit:
		c.checkCompositeLit(e)
	case *ast.KeyValueExpr:
		c.checkExpr(e.Value)
	case *ast.StarExpr:
		c.refuse(e.Pos(), "a pointer dereference",
			"pass the value; the subset has no pointers of its own")
	case *ast.SliceExpr:
		c.refuse(e.Pos(), "a slice expression",
			"index the fixed-length array, or pass an offset and a count")
	case *ast.TypeAssertExpr:
		c.refuse(e.Pos(), "a type assertion",
			"the subset has no interfaces; switch on an int32 tag field instead")
	case *ast.FuncLit:
		c.refuse(e.Pos(), "a function literal",
			"declare a named function in this package and call it; SourcePawn has no closures")
	case *ast.ArrayType, *ast.MapType, *ast.ChanType, *ast.InterfaceType, *ast.StructType, *ast.FuncType:
		c.checkType(expr)
	default:
		c.refuse(expr.Pos(), fmt.Sprintf("the expression %T", expr),
			"the subset has identifiers, numeric literals, arithmetic, indexing, field access and calls")
	}
}

func (c *checker) checkBasicLit(lit *ast.BasicLit) {
	switch lit.Kind {
	case token.STRING:
		c.refuse(lit.Pos(), "a string literal",
			"the subset has no strings; use an int32 identifier and keep the name table in internal/tables")
	case token.IMAG:
		c.refuse(lit.Pos(), "an imaginary literal",
			"SourcePawn has no complex numbers")
	}
}

func (c *checker) checkBinary(e *ast.BinaryExpr) {
	if e.Op == token.AND_NOT {
		c.refuse(e.OpPos, "the &^ operator",
			"SourcePawn has no AND NOT; write x & ^y")
	}
	c.checkExpr(e.X)
	c.checkExpr(e.Y)
}

func (c *checker) checkUnary(e *ast.UnaryExpr) {
	switch e.Op {
	case token.AND:
		c.refuse(e.OpPos, "taking an address",
			"pass the value; several results become by-reference parameters in the generated function")
		return
	case token.ARROW:
		c.refuse(e.OpPos, "a channel receive",
			"the subset has no channels; remove it")
		return
	}
	c.checkExpr(e.X)
}

func (c *checker) checkCompositeLit(e *ast.CompositeLit) {
	if e.Type != nil {
		c.checkType(e.Type)
	}
	for _, elt := range e.Elts {
		c.checkExpr(elt)
	}
}

func (c *checker) checkSelector(e *ast.SelectorExpr) {
	id, isIdent := e.X.(*ast.Ident)
	if !isIdent {
		c.checkExpr(e.X)
		return
	}
	path, isImport := c.imports[id.Name]
	if !isImport {
		return // a struct field, checked by the type declaration it came from
	}
	if !c.packages[path][e.Sel.Name] {
		c.refuse(e.Sel.Pos(), fmt.Sprintf("%s.%s, which the generator has no mapping for", id.Name, e.Sel.Name),
			"use one of the mapped identifiers, or move the work into a native binding")
	}
}
