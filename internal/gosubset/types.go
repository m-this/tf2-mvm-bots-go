package gosubset

import (
	"fmt"
	"go/ast"
)

// basicTypes are the value types a SourcePawn cell or enum struct field can
// hold. byte and rune are the aliases of uint8 and int32.
var basicTypes = map[string]bool{
	"bool": true,
	"int8": true, "int16": true, "int32": true, "int64": true,
	"uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"byte": true, "rune": true,
	"float32": true, "float64": true,
}

// refusedTypes carries the fix for the types a reader is most likely to write.
var refusedTypes = map[string]string{
	"int":        "a SourcePawn cell is 32 bits wide; write int32 so the width is in the source",
	"uint":       "a SourcePawn cell is 32 bits wide; write uint32 so the width is in the source",
	"uintptr":    "a generated body never holds an address; pass the value itself",
	"string":     "the subset has no strings; pass an int32 identifier and keep the name table in internal/tables",
	"error":      "return a bool or a sentinel int32 the caller switches on; SourcePawn has no error interface",
	"any":        "name the concrete type; the generator emits one function per type, not a boxed value",
	"complex64":  "SourcePawn has no complex numbers; use two float32 fields",
	"complex128": "SourcePawn has no complex numbers; use two float32 fields",
}

func (c *checker) checkType(expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.Ident:
		c.checkTypeName(t)
	case *ast.ArrayType:
		c.checkArrayType(t)
	case *ast.StructType:
		c.refuse(t.Pos(), "an anonymous struct type",
			"declare a named struct at package level; a generated enum struct needs a name")
	case *ast.StarExpr:
		c.refuse(t.Pos(), "a pointer type",
			"pass the value; multiple results become by-reference parameters in the generated function, and nothing else takes an address")
	case *ast.MapType:
		c.refuse(t.Pos(), "a map type",
			"use a fixed-length array indexed by a small int32, sized by a constant")
	case *ast.ChanType:
		c.refuse(t.Pos(), "a channel type",
			"a generated body is one call on one thread; remove the channel")
	case *ast.InterfaceType:
		c.refuse(t.Pos(), "an interface type",
			"name the concrete type; SourcePawn dispatch is a switch, written out")
	case *ast.FuncType:
		c.refuse(t.Pos(), "a function type",
			"call the function by name; the generator has no function values")
	case *ast.Ellipsis:
		c.refuse(t.Pos(), "a variadic parameter",
			"take a fixed-length array and a count")
	case *ast.SelectorExpr:
		// A type from the extern package is a SourcePawn tag that
		// already exists, named there so a ported signature keeps it.
		// Anything else has nothing for the generator to emit.
		c.checkSelector(t)
	case *ast.IndexExpr, *ast.IndexListExpr:
		c.refuse(expr.Pos(), "a generic type instantiation",
			"write the concrete type; SourcePawn has no type parameters")
	default:
		c.refuse(expr.Pos(), "an unrecognised type expression",
			"the subset has sized integers, float32, float64, bool, fixed-length arrays and named structs of those")
	}
}

func (c *checker) checkTypeName(id *ast.Ident) {
	if basicTypes[id.Name] || c.types[id.Name] {
		return
	}
	if fix, known := refusedTypes[id.Name]; known {
		c.refuse(id.Pos(), fmt.Sprintf("the type %s", id.Name), fix)
		return
	}
	c.refuse(id.Pos(), fmt.Sprintf("the unknown type %s", id.Name),
		"declare the type in this package, or use a sized numeric type, float32, float64 or bool")
}

func (c *checker) checkArrayType(t *ast.ArrayType) {
	if t.Len == nil {
		c.refuse(t.Pos(), "a slice type",
			"use a fixed-length array sized by a constant, plus a count parameter for how much of it is filled")
		return
	}
	if _, isEllipsis := t.Len.(*ast.Ellipsis); isEllipsis {
		c.refuse(t.Len.Pos(), "an array length written as [...]",
			"write the length as a constant so the generated declaration has one")
		return
	}
	c.checkConstExpr(t.Len, "an array length")
	c.checkType(t.Elt)
}

// checkConstExpr accepts what SourcePawn will accept where it needs a compile
// time constant: literals, named constants and arithmetic over them.
func (c *checker) checkConstExpr(expr ast.Expr, what string) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		c.checkBasicLit(e)
	case *ast.Ident:
		return
	case *ast.ParenExpr:
		c.checkConstExpr(e.X, what)
	case *ast.UnaryExpr:
		c.checkConstExpr(e.X, what)
	case *ast.BinaryExpr:
		c.checkConstExpr(e.X, what)
		c.checkConstExpr(e.Y, what)
	default:
		c.refuse(expr.Pos(), what+" that is not a constant",
			"use a literal or a named constant; SourcePawn resolves it at compile time")
	}
}
