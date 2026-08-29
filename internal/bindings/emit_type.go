package bindings

import (
	"fmt"
	"strconv"
	"strings"
)

// baseTypes maps the SourcePawn primitive tags onto Go. Every other tag name
// is a methodmap, enum or typedef and is emitted under its own name.
var baseTypes = map[string]string{
	"int":   "int32",
	"float": "float32",
	"bool":  "bool",
	"char":  "byte",
	"any":   "int32",
	"void":  "",
}

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// goIdent makes a SourcePawn identifier safe to use in Go without renaming
// anything that does not have to be renamed.
func goIdent(name string) string {
	if name == "" {
		return ""
	}
	if goKeywords[name] {
		return name + "_"
	}
	return strings.ReplaceAll(name, "@", "_")
}

// goType renders a parameter, field or return type. It returns an error
// rather than a guess for anything Go cannot express.
func goType(t Type) (string, error) {
	base, ok := baseTypes[t.Name]
	if !ok {
		base = goIdent(t.Name)
	}
	if base == "" && len(t.Dims) == 0 && !t.ByRef {
		return "", nil
	}
	if t.Name == "void" {
		return "", fmt.Errorf("void used as a value type")
	}
	if t.Name == "char" && len(t.Dims) == 1 && t.Dims[0] == "" {
		if t.Const {
			return "string", nil
		}
		return "[]byte", nil
	}
	rendered := base
	for i := len(t.Dims) - 1; i >= 0; i-- {
		rendered = arrayPrefix(t.Dims[i]) + rendered
	}
	if t.ByRef {
		if len(t.Dims) > 0 {
			return "", fmt.Errorf("by-reference array parameter")
		}
		rendered = "*" + rendered
	}
	return rendered, nil
}

// arrayPrefix renders one dimension. A dimension whose size is not a plain
// integer literal becomes a slice: the size is a SourcePawn constant that
// only the SourcePawn side can resolve.
func arrayPrefix(dim string) string {
	if n, err := strconv.Atoi(dim); err == nil && n > 0 {
		return "[" + strconv.Itoa(n) + "]"
	}
	return "[]"
}

// goParamType renders a parameter type. SourcePawn passes every array by
// reference, so a non-const fixed-size array parameter becomes a pointer:
// passing a Go array by value would silently drop what the callee wrote.
// Slices and strings already alias their backing store and stay as they are.
func goParamType(t Type) (string, error) {
	rendered, err := goType(t)
	if err != nil {
		return "", err
	}
	if t.Const || len(t.Dims) == 0 || !strings.HasPrefix(rendered, "[") || strings.HasPrefix(rendered, "[]") {
		return rendered, nil
	}
	return "*" + rendered, nil
}

// signature renders a Go parameter list and result for one native, method or
// typedef, or refuses with a reason.
func signature(params []Param, ret Type) (string, string, error) {
	var b strings.Builder
	for i, pm := range params {
		if pm.Variadic {
			return "", "", fmt.Errorf("variadic parameter %q", pm.Name)
		}
		typ, err := goParamType(pm.Type)
		if err != nil {
			return "", "", fmt.Errorf("parameter %q: %w", pm.Name, err)
		}
		if typ == "" {
			return "", "", fmt.Errorf("parameter %q has no type", pm.Name)
		}
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s %s", paramName(pm, i), typ)
	}
	if len(ret.Dims) > 0 {
		return "", "", fmt.Errorf("return type is an array")
	}
	result, err := goType(ret)
	if err != nil {
		return "", "", fmt.Errorf("return type: %w", err)
	}
	return b.String(), result, nil
}

func paramName(pm Param, i int) string {
	if pm.Name == "" {
		return "arg" + strconv.Itoa(i)
	}
	return goIdent(pm.Name)
}

// defaultsNote records the SourcePawn default arguments that Go cannot
// express, so a reader of the generated file is not misled into thinking the
// parameters are all required at the call site in SourcePawn.
func defaultsNote(params []Param) string {
	var parts []string
	for _, pm := range params {
		if pm.Default != "" {
			parts = append(parts, paramName(pm, 0)+"="+pm.Default)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "// SourcePawn defaults: " + strings.Join(parts, ", ")
}
