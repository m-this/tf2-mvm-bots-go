package bindings

import "strings"

func (p *parser) enum() {
	start := p.cur()
	p.advance() // "enum"
	e := Enum{Doc: start.doc, Pos: Pos{p.name, start.line}}
	if p.cur().kind == tokIdent {
		e.Name = p.cur().text
		p.advance()
	}
	if p.isPunct("(") {
		p.advance()
		var b strings.Builder
		for !p.done() && !p.isPunct(")") {
			b.WriteString(p.cur().text)
			p.advance()
		}
		p.accept(")")
		e.Increment = strings.TrimSpace(b.String())
	}
	p.accept(":")
	if !p.accept("{") {
		p.refuse(e.Pos, "enum", p.textFrom(start.line), "missing body")
		p.skipDeclaration()
		return
	}
	for range maxMembers {
		if p.done() || p.accept("}") {
			p.accept(";")
			p.file.Enums = append(p.file.Enums, e)
			return
		}
		if p.cur().kind == tokDirective {
			p.advance()
			continue
		}
		if p.cur().kind != tokIdent {
			p.refuse(p.pos(), "enum", p.textFrom(p.cur().line), "unreadable entry in enum "+e.Name)
			p.advance()
			continue
		}
		entry := EnumEntry{Name: p.cur().text, Pos: p.pos()}
		p.advance()
		if p.isPunct("[") {
			p.parseDims() // legacy sized enum cell; the size is not part of the value
		}
		if p.accept("=") {
			entry.Value = p.parseEnumValue()
		}
		e.Entries = append(e.Entries, entry)
		p.accept(",")
	}
	p.file.Enums = append(p.file.Enums, e)
}

func (p *parser) parseEnumValue() string {
	var b strings.Builder
	depth := 0
	for !p.done() {
		switch {
		case p.isPunct("("):
			depth++
		case p.isPunct(")"):
			depth--
		case p.isPunct(",") && depth == 0:
			return strings.TrimSpace(b.String())
		case p.isPunct("}") && depth == 0:
			return strings.TrimSpace(b.String())
		}
		b.WriteString(p.cur().text)
		p.advance()
	}
	return strings.TrimSpace(b.String())
}

func (p *parser) enumStruct() {
	start := p.cur()
	p.advance() // "enum"
	p.advance() // "struct"
	if p.cur().kind != tokIdent {
		p.refuse(Pos{p.name, start.line}, "enum struct", p.textFrom(start.line), "missing name")
		p.skipDeclaration()
		return
	}
	es := EnumStruct{Name: p.cur().text, Doc: start.doc, Pos: Pos{p.name, start.line}}
	p.advance()
	if !p.accept("{") {
		p.refuse(es.Pos, "enum struct", p.textFrom(start.line), "missing body")
		p.skipDeclaration()
		return
	}
	for range maxMembers {
		if p.done() || p.accept("}") {
			p.accept(";")
			p.file.EnumStructs = append(p.file.EnumStructs, es)
			return
		}
		before := p.at
		p.enumStructMember(&es)
		if p.at == before {
			p.advance()
		}
	}
	p.file.EnumStructs = append(p.file.EnumStructs, es)
}

func (p *parser) enumStructMember(es *EnumStruct) {
	if p.cur().kind == tokDirective {
		p.advance()
		return
	}
	if p.isIdent("public") {
		mm := Methodmap{Name: es.Name}
		p.member(&mm)
		es.Methods = append(es.Methods, mm.Methods...)
		return
	}
	start := p.cur()
	typ, ok := p.parseReturnType()
	if !ok || p.cur().kind != tokIdent {
		p.refuse(Pos{p.name, start.line}, "enum struct field", p.textFrom(start.line), "unreadable field in "+es.Name)
		p.skipDeclaration()
		return
	}
	name := p.cur().text
	p.advance()
	if p.isPunct("(") {
		params, ok := p.parseParams()
		if !ok {
			p.refuse(Pos{p.name, start.line}, "enum struct method", p.textFrom(start.line), "unreadable parameter list in "+es.Name)
			p.skipDeclaration()
			return
		}
		p.skipBody()
		es.Methods = append(es.Methods, Method{
			Name: name, Return: typ, Params: params,
			Doc: start.doc, Pos: Pos{p.name, start.line},
		})
		return
	}
	typ.Dims = append(typ.Dims, p.parseDims()...)
	p.accept(";")
	es.Fields = append(es.Fields, Field{Name: name, Type: typ, Pos: Pos{p.name, start.line}})
}

func (p *parser) typedef() {
	start := p.cur()
	p.advance() // "typedef"
	if p.cur().kind != tokIdent {
		p.refuse(Pos{p.name, start.line}, "typedef", p.textFrom(start.line), "missing name")
		p.skipDeclaration()
		return
	}
	td := Typedef{Name: p.cur().text, Doc: start.doc, Pos: Pos{p.name, start.line}}
	p.advance()
	if !p.accept("=") || !p.accept("function") {
		p.refuse(td.Pos, "typedef", p.textFrom(start.line), "not a function typedef")
		p.skipDeclaration()
		return
	}
	ret, ok := p.parseReturnType()
	if !ok {
		p.refuse(td.Pos, "typedef", p.textFrom(start.line), "unreadable return type")
		p.skipDeclaration()
		return
	}
	td.Return = ret
	params, ok := p.parseParams()
	if !ok {
		p.refuse(td.Pos, "typedef", p.textFrom(start.line), "unreadable parameter list")
		p.skipDeclaration()
		return
	}
	td.Params = params
	p.accept(";")
	p.file.Typedefs = append(p.file.Typedefs, td)
}

// typeset declares one name with several call signatures. Go has no such
// type, so the name is refused rather than collapsed onto one signature.
func (p *parser) typeset() {
	start := p.cur()
	p.advance()
	name := ""
	if p.cur().kind == tokIdent {
		name = p.cur().text
	}
	p.refuse(Pos{p.name, start.line}, "typeset", name, "typeset has several signatures under one name")
	p.skipDeclaration()
}
