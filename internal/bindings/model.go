// Package bindings parses SourcePawn include files and emits Go declarations
// for the natives, methodmaps, enums and constants they declare.
//
// The parser is deliberately partial: it understands declarations and nothing
// else. Function bodies, expressions and the preprocessor are skipped, not
// interpreted. Anything it cannot read exactly is recorded as a Refusal rather
// than guessed at.
package bindings

import "fmt"

// Pos is a location in an include file.
type Pos struct {
	File string
	Line int32
}

func (p Pos) String() string { return fmt.Sprintf("%s:%d", p.File, p.Line) }

// Type is a SourcePawn type as it appears in a declaration: a tag name plus
// the decorations the declaration syntax allows. Dims holds one entry per
// array dimension, in source order, each the raw text of the size expression
// or "" when the dimension is unsized.
type Type struct {
	Name  string
	Const bool
	ByRef bool
	Dims  []string
}

// IsVoid reports whether the type is the absence of a value.
func (t Type) IsVoid() bool { return t.Name == "void" }

// Param is one formal parameter of a native, method or callback signature.
type Param struct {
	Name     string
	Type     Type
	Default  string // raw expression text, empty when the parameter is required
	Variadic bool
}

// Native is a free-standing function declaration: a `native`, or a `stock`
// whose SourcePawn body the include ships. Both are callable API and both
// need a Go declaration; Stock says which one it was.
type Native struct {
	Stock  bool
	Name   string
	Return Type
	Params []Param
	Doc    string
	Pos    Pos
}

// MethodKind separates the three shapes a methodmap member can take.
type MethodKind uint8

// The three shapes: an ordinary method on an instance, a constructor named
// after its methodmap, and a static method with no instance.
const (
	MethodPlain MethodKind = iota
	MethodConstructor
	MethodStatic
)

// Method is a member function of a methodmap or an enum struct.
type Method struct {
	Name   string
	Kind   MethodKind
	Return Type
	Params []Param
	Native bool // false when the include ships a SourcePawn body for it
	Doc    string
	Pos    Pos
}

// Property is a methodmap property. A property has a getter, a setter, or both.
type Property struct {
	Name      string
	Type      Type
	Get       bool
	Set       bool
	GetNative bool
	SetNative bool
	Doc       string
	Pos       Pos
}

// Methodmap is a methodmap block, including its inheritance edge.
type Methodmap struct {
	Name       string
	Parent     string // empty when the methodmap has no base
	Nullable   bool
	Methods    []Method
	Properties []Property
	Doc        string
	Pos        Pos
}

// EnumEntry is one member of an enum. Value holds the raw text of the
// initialiser, empty when the entry takes the implicit next value.
type EnumEntry struct {
	Name  string
	Value string
	Pos   Pos
}

// Enum is an enum block. Name is empty for an anonymous enum. Increment holds
// the raw text of the `enum (<<= 1)` style step expression, empty when absent.
type Enum struct {
	Name      string
	Increment string
	Entries   []EnumEntry
	Doc       string
	Pos       Pos
}

// Field is one member of an enum struct.
type Field struct {
	Name string
	Type Type
	Pos  Pos
}

// EnumStruct is an `enum struct` block.
type EnumStruct struct {
	Name    string
	Fields  []Field
	Methods []Method
	Doc     string
	Pos     Pos
}

// Define is an object-like preprocessor constant. Value is the raw
// replacement text with comments stripped.
type Define struct {
	Name  string
	Value string
	Pos   Pos
}

// Typedef is a `typedef X = function ret (params);` callback type.
type Typedef struct {
	Name   string
	Return Type
	Params []Param
	Doc    string
	Pos    Pos
}

// Refusal records one declaration the parser or the emitter would not handle.
// A refusal is never a warning to be filtered out: it is the list of things a
// caller must either fix here or write by hand.
type Refusal struct {
	Pos    Pos
	Kind   string // "native", "methodmap", "enum", "define", "declaration", ...
	Detail string // the offending source text, truncated
	Reason string
}

func (r Refusal) String() string {
	return fmt.Sprintf("%s: %s: %s (%s)", r.Pos, r.Kind, r.Reason, r.Detail)
}

// File is everything one include file declares.
type File struct {
	Path        string
	Natives     []Native
	Stocks      []Native
	Methodmaps  []Methodmap
	Enums       []Enum
	EnumStructs []EnumStruct
	Defines     []Define
	Typedefs    []Typedef
	Refusals    []Refusal
}
