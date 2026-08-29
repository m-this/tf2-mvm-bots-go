package bindings

// maxMembers bounds every member loop below. The largest block in the tree,
// CBaseEntity, has under 400 members.
const maxMembers = 4096

func (p *parser) methodmap() {
	start := p.cur()
	p.advance() // "methodmap"
	if p.cur().kind != tokIdent {
		p.refuse(Pos{p.name, start.line}, "methodmap", p.textFrom(start.line), "missing name")
		p.skipDeclaration()
		return
	}
	mm := Methodmap{Name: p.cur().text, Doc: start.doc, Pos: Pos{p.name, start.line}}
	p.advance()
	for p.isPunct("<") || p.isIdent("__nullable__") {
		if p.accept("__nullable__") {
			mm.Nullable = true
			continue
		}
		p.advance() // "<"
		if p.cur().kind != tokIdent {
			p.refuse(mm.Pos, "methodmap", p.textFrom(start.line), "unreadable base type")
			p.skipDeclaration()
			return
		}
		mm.Parent = p.cur().text
		p.advance()
	}
	if !p.accept("{") {
		p.refuse(mm.Pos, "methodmap", p.textFrom(start.line), "missing body")
		p.skipDeclaration()
		return
	}
	p.methodmapBody(&mm)
	p.file.Methodmaps = append(p.file.Methodmaps, mm)
}

func (p *parser) methodmapBody(mm *Methodmap) {
	for range maxMembers {
		if p.done() || p.accept("}") {
			p.accept(";")
			return
		}
		before := p.at
		switch {
		case p.cur().kind == tokDirective:
			p.advance()
		case p.isIdent("property"):
			p.property(mm)
		case p.isIdent("public"):
			p.member(mm)
		default:
			p.refuse(p.pos(), "methodmap member", p.textFrom(p.cur().line), "unrecognised member in "+mm.Name)
			p.skipDeclaration()
		}
		if p.at == before {
			p.advance()
		}
	}
}

// member parses `public [static] [native] <type> <name>(params)` plus either a
// semicolon or a SourcePawn body.
func (p *parser) member(mm *Methodmap) {
	start := p.cur()
	p.advance() // "public"
	kind := MethodPlain
	if p.accept("static") {
		kind = MethodStatic
	}
	if p.isIdent("native") && p.peekAt(1).kind == tokPunct && p.peekAt(1).text == "~" {
		p.refuse(Pos{p.name, start.line}, "methodmap member", p.textFrom(start.line),
			"destructor has no Go equivalent")
		p.skipDeclaration()
		return
	}
	if p.isIdent("native") {
		p.native(mm)
		if kind == MethodStatic && len(mm.Methods) > 0 {
			mm.Methods[len(mm.Methods)-1].Kind = MethodStatic
		}
		return
	}
	ret, ok := p.parseReturnType()
	if !ok {
		p.refuse(Pos{p.name, start.line}, "method", p.textFrom(start.line), "unreadable return type")
		p.skipDeclaration()
		return
	}
	name := ret.Name
	if p.cur().kind == tokIdent {
		name = p.cur().text
		p.advance()
	} else {
		ret = Type{Name: mm.Name}
		kind = MethodConstructor
	}
	params, ok := p.parseParams()
	if !ok {
		p.refuse(Pos{p.name, start.line}, "method", p.textFrom(start.line), "unreadable parameter list")
		p.skipDeclaration()
		return
	}
	if name == mm.Name {
		kind = MethodConstructor
	}
	p.skipBody()
	mm.Methods = append(mm.Methods, Method{
		Name: name, Kind: kind, Return: ret, Params: params, Native: false,
		Doc: start.doc, Pos: Pos{p.name, start.line},
	})
}

// skipBody consumes either a trailing semicolon or a balanced brace block.
func (p *parser) skipBody() {
	if p.accept(";") {
		return
	}
	if !p.isPunct("{") {
		return
	}
	depth := 0
	for !p.done() {
		if p.isPunct("{") {
			depth++
		}
		if p.isPunct("}") {
			depth--
			if depth == 0 {
				p.advance()
				p.accept(";")
				return
			}
		}
		p.advance()
	}
}

func (p *parser) property(mm *Methodmap) {
	start := p.cur()
	p.advance() // "property"
	typ, ok := p.parseReturnType()
	if !ok || p.cur().kind != tokIdent {
		p.refuse(Pos{p.name, start.line}, "property", p.textFrom(start.line), "unreadable property header")
		p.skipDeclaration()
		return
	}
	prop := Property{Name: p.cur().text, Type: typ, Doc: start.doc, Pos: Pos{p.name, start.line}}
	p.advance()
	if !p.accept("{") {
		p.refuse(prop.Pos, "property", p.textFrom(start.line), "missing accessor block")
		p.skipDeclaration()
		return
	}
	for range maxMembers {
		if p.done() || p.accept("}") {
			p.accept(";")
			mm.Properties = append(mm.Properties, prop)
			return
		}
		if p.cur().kind == tokDirective {
			p.advance()
			continue
		}
		before := p.at
		p.accessor(&prop)
		if p.at == before {
			p.advance()
		}
	}
	mm.Properties = append(mm.Properties, prop)
}

func (p *parser) accessor(prop *Property) {
	if !p.accept("public") {
		p.refuse(p.pos(), "property", p.textFrom(p.cur().line), "unrecognised accessor for "+prop.Name)
		p.skipDeclaration()
		return
	}
	isNative := p.accept("native")
	switch {
	case p.isIdent("get"):
		prop.Get, prop.GetNative = true, isNative
	case p.isIdent("set"):
		prop.Set, prop.SetNative = true, isNative
	default:
		p.refuse(p.pos(), "property", p.textFrom(p.cur().line), "accessor is neither get nor set")
		p.skipDeclaration()
		return
	}
	p.advance()
	if _, ok := p.parseParams(); !ok {
		p.refuse(p.pos(), "property", p.textFrom(p.cur().line), "unreadable accessor parameters")
	}
	p.skipBody()
}
