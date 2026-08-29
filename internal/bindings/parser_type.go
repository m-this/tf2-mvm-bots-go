package bindings

import "strings"

// maxParams bounds the parameter loop. SourcePawn itself caps a call at 127
// arguments; nothing in the tree comes close.
const maxParams = 128

// parseDims consumes a run of `[expr]` and returns one entry per dimension,
// each the raw text of the size expression or "" when unsized.
func (p *parser) parseDims() []string {
	var dims []string
	for p.isPunct("[") {
		p.advance()
		var b strings.Builder
		depth := 1
		for !p.done() {
			if p.isPunct("[") {
				depth++
			}
			if p.isPunct("]") {
				depth--
				if depth == 0 {
					break
				}
			}
			b.WriteString(p.cur().text)
			p.advance()
		}
		p.accept("]")
		dims = append(dims, strings.TrimSpace(b.String()))
	}
	return dims
}

// parseReturnType reads the type in front of a native or method name.
func (p *parser) parseReturnType() (Type, bool) {
	var t Type
	if p.isIdent("const") {
		t.Const = true
		p.advance()
	}
	if p.cur().kind != tokIdent {
		return t, false
	}
	t.Name = p.cur().text
	p.advance()
	t.Dims = p.parseDims()
	return t, true
}

// parseParams reads a parenthesised formal parameter list.
func (p *parser) parseParams() ([]Param, bool) {
	if !p.isPunct("(") {
		return nil, false
	}
	p.advance()
	var params []Param
	for len(params) < maxParams {
		if p.isPunct(")") {
			p.advance()
			return params, true
		}
		if p.done() {
			return nil, false
		}
		param, ok := p.parseParam()
		if !ok {
			return nil, false
		}
		params = append(params, param)
		if !p.accept(",") && !p.isPunct(")") {
			return nil, false
		}
	}
	return nil, false
}

func (p *parser) parseParam() (Param, bool) {
	var pm Param
	if p.isPunct("...") {
		p.advance()
		pm.Variadic = true
		pm.Name = "args"
		return pm, true
	}
	if p.isIdent("const") {
		pm.Type.Const = true
		p.advance()
	}
	if p.cur().kind != tokIdent {
		return pm, false
	}
	pm.Type.Name = p.cur().text
	p.advance()
	pm.Type.Dims = p.parseDims()
	if p.isPunct("&") {
		pm.Type.ByRef = true
		p.advance()
	}
	if p.isPunct("...") {
		p.advance()
		pm.Variadic = true
		pm.Name = "args"
		return pm, true
	}
	if p.cur().kind != tokIdent {
		// A lone identifier is SourcePawn's untagged parameter: the name is
		// what was read as the type, and the type is implicitly `any`.
		if len(pm.Type.Dims) == 0 && !pm.Type.ByRef && (p.isPunct("=") || p.isPunct(",") || p.isPunct(")")) {
			pm.Name, pm.Type.Name = pm.Type.Name, "any"
		} else {
			return pm, false
		}
	} else {
		pm.Name = p.cur().text
		p.advance()
	}
	pm.Type.Dims = append(pm.Type.Dims, p.parseDims()...)
	if p.isPunct("=") {
		p.advance()
		pm.Default = p.parseDefault()
	}
	return pm, true
}

// parseDefault reads a default-value expression up to the comma or closing
// paren that ends the parameter, keeping the raw text.
func (p *parser) parseDefault() string {
	var b strings.Builder
	depth := 0
	for !p.done() {
		switch {
		case p.isPunct("(") || p.isPunct("[") || p.isPunct("{"):
			depth++
		case p.isPunct(")") || p.isPunct("]") || p.isPunct("}"):
			if depth == 0 {
				return strings.TrimSpace(b.String())
			}
			depth--
		case p.isPunct(",") && depth == 0:
			return strings.TrimSpace(b.String())
		}
		b.WriteString(p.cur().text)
		p.advance()
	}
	return strings.TrimSpace(b.String())
}
