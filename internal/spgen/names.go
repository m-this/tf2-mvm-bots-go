package spgen

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
)

// reserved are the SourcePawn keywords and built-in tags a Go identifier is
// allowed to be. A local named class or new is renamed rather than refused,
// because the Go side should read as Go.
var reserved = map[string]bool{
	"any": true, "assert": true, "bool": true, "break": true, "case": true,
	"cast_to": true, "cellsof": true, "char": true, "class": true, "const": true,
	"continue": true, "decl": true, "default": true, "defined": true, "delete": true,
	"do": true, "else": true, "enum": true, "false": true, "float": true, "for": true,
	"forward": true, "funcenum": true, "function": true, "functag": true, "goto": true,
	"if": true, "int": true, "methodmap": true, "native": true, "new": true, "null": true,
	"object": true, "operator": true, "property": true, "public": true, "return": true,
	"sizeof": true, "static": true, "stock": true, "struct": true, "switch": true,
	"tagof": true, "this": true, "true": true, "typedef": true, "typeset": true,
	"union": true, "using": true, "view_as": true, "void": true, "while": true,
	"Float": true, "String": true, "Handle": true, "Function": true, "Action": true,
}

// identifier is a local name: a SourcePawn keyword gets a trailing underscore
// so `class` survives the trip.
func identifier(goName string) string {
	if reserved[goName] {
		return goName + "_"
	}
	return goName
}

// name is a package-level name. Every one of them carries the prefix, because
// Action, RoundState and Address are SourceMod's names already.
func (g *generator) name(goName string) string {
	return g.cfg.Prefix + strings.ToUpper(goName[:1]) + goName[1:]
}

// spType is the SourcePawn tag for a Go type.
func (g *generator) spType(t types.Type, pos token.Pos) string {
	switch u := t.(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Bool, types.UntypedBool:
			return "bool"
		// types.Int is here for len, which is the only way an untyped int
		// constant reaches this: gosubset refuses `int` written in source.
		case types.Int32, types.Uint32, types.Int16, types.Uint16, types.Int8, types.Uint8,
			types.Int, types.Uint, types.UntypedInt, types.UntypedRune:
			return "int"
		case types.Float32, types.UntypedFloat:
			return "float"
		}
		g.refuse(pos, fmt.Sprintf("the type %s, which does not fit a 32-bit cell", u.Name()),
			"write int32, float32 or bool")
		return "int"
	case *types.Named:
		return g.name(u.Obj().Name())
	case *types.Array:
		g.refuse(pos, "an array where a single cell is needed",
			"pass the array as its own parameter")
		return "int"
	}
	g.refuse(pos, fmt.Sprintf("the type %s", t.String()), "the subset has sized integers, float32, bool, fixed arrays and named structs")
	return "int"
}

// declare is a declaration: the tag, the name, and the array bound after it,
// which is where SourcePawn puts it.
func (g *generator) declare(t types.Type, name string, pos token.Pos) string {
	if arr, ok := t.(*types.Array); ok {
		return fmt.Sprintf("%s %s[%d]", g.spType(arr.Elem(), pos), name, arr.Len())
	}
	return g.spType(t, pos) + " " + name
}
