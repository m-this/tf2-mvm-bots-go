package bindings

import (
	"fmt"
	"strconv"
	"strings"
)

// enum emits the enum's type, when it is named, and its constants. Values are
// resolved left to right; the first entry whose value cannot be resolved ends
// the enum, because every entry after it depends on that value.
func (e *emitter) enum(en Enum) {
	typeName := goIdent(en.Name)
	step, ok := enumStep(en.Increment)
	if !ok {
		e.refuse(en.Pos, "enum", en.Name+" ("+en.Increment+")", "unsupported enum increment")
		return
	}
	// A methodmap of the same name owns the type: SourcePawn declares the
	// tag with an enum and hangs the methods off it. The entries are then
	// values of that type, not constants of an int32 one.
	asMethodmap := e.mmNames[en.Name]
	if typeName != "" && !asMethodmap {
		if !e.claim("", typeName, en.Pos) {
			return
		}
		e.doc(en.Doc)
		fmt.Fprintf(&e.b, "type %s int32\n\n", typeName)
	}
	var next int64
	var lines []string
	for i, entry := range en.Entries {
		value, err := entryValue(entry, e.consts, next, i == 0)
		if err != nil {
			e.refuse(entry.Pos, "enum entry", en.Name+"."+entry.Name+" = "+entry.Value,
				err.Error())
			break
		}
		e.consts[entry.Name] = value
		next = step(value)
		if !e.claim("", goIdent(entry.Name), entry.Pos) {
			continue
		}
		if asMethodmap {
			// The tag's only enumerated value is the invalid one, which is
			// the zero handle. A non-zero value would have to name a field
			// the methodmap may inherit rather than declare, so it is
			// refused instead of guessed at.
			if value != 0 {
				e.refuse(entry.Pos, "enum entry", en.Name+"."+entry.Name,
					"non-zero value on a tag that a methodmap owns")
				continue
			}
			lines = append(lines, fmt.Sprintf("var %s %s\n\n", goIdent(entry.Name), typeName))
			continue
		}
		lines = append(lines, fmt.Sprintf("\t%s %s = %d\n", goIdent(entry.Name), typeName, value))
	}
	if len(lines) == 0 {
		return
	}
	if asMethodmap {
		for _, l := range lines {
			e.b.WriteString(l)
		}
		return
	}
	e.b.WriteString("const (\n")
	for _, l := range lines {
		e.b.WriteString(l)
	}
	e.b.WriteString(")\n\n")
}

// enumStep turns the `enum (<<= 1)` style header into the function that
// produces the next implicit value.
func enumStep(increment string) (func(int64) int64, bool) {
	increment = strings.TrimSpace(increment)
	if increment == "" {
		return func(v int64) int64 { return v + 1 }, true
	}
	if rest, found := strings.CutPrefix(increment, "<<="); found {
		// A bit size of 6 caps the shift at 63, which is what an int64 can
		// take without the shift being undefined.
		shift, err := strconv.ParseUint(strings.TrimSpace(rest), 0, 6)
		if err != nil {
			return nil, false
		}
		return func(v int64) int64 { return v << shift }, true
	}
	if rest, found := strings.CutPrefix(increment, "+="); found {
		n, err := strconv.ParseInt(strings.TrimSpace(rest), 0, 64)
		if err != nil {
			return nil, false
		}
		return func(v int64) int64 { return v + n }, true
	}
	if rest, found := strings.CutPrefix(increment, "*="); found {
		n, err := strconv.ParseInt(strings.TrimSpace(rest), 0, 64)
		if err != nil {
			return nil, false
		}
		return func(v int64) int64 { return v * n }, true
	}
	return nil, false
}

// entryValue resolves one entry: an explicit literal, a reference to an
// earlier entry of the same enum, or the implicit next value.
func entryValue(entry EnumEntry, known map[string]int64, next int64, first bool) (int64, error) {
	if entry.Value == "" {
		if first {
			return 0, nil
		}
		return next, nil
	}
	return evalConst(entry.Value, known)
}

// define emits a `#define` as a Go constant when its replacement text is a
// single literal. Anything else is an expression over other SourcePawn
// constants, which only the SourcePawn compiler can evaluate.
func (e *emitter) define(d Define) {
	value := strings.TrimSpace(d.Value)
	switch {
	case value == "":
		return // an include guard, not a constant
	case strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) && len(value) >= 2:
		if !e.claim("", goIdent(d.Name), d.Pos) {
			return
		}
		fmt.Fprintf(&e.b, "const %s = %s\n\n", goIdent(d.Name), value)
	default:
		if v, err := evalConst(value, e.consts); err == nil {
			e.consts[d.Name] = v
			if !e.claim("", goIdent(d.Name), d.Pos) {
				return
			}
			fmt.Fprintf(&e.b, "const %s = %d\n\n", goIdent(d.Name), v)
			return
		}
		if f, err := strconv.ParseFloat(strings.Trim(value, "()"), 64); err == nil {
			fmt.Fprintf(&e.b, "const %s = %v\n\n", goIdent(d.Name), f)
			return
		}
		e.refuse(d.Pos, "define", d.Name+" "+value, "replacement text is not a constant expression")
	}
}
