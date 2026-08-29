package bindings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParseFile reads and parses one include file.
func ParseFile(path string) (*File, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading include: %w", err)
	}
	return Parse(path, src), nil
}

// Parse parses one include file already in memory. It never fails: everything
// it cannot read lands in File.Refusals.
func Parse(path string, src []byte) *File {
	p := &parser{
		file: &File{Path: path},
		toks: lex(src),
		name: filepath.Base(path),
	}
	p.run()
	return p.file
}

type parser struct {
	file *File
	toks []token
	at   int
	name string
}

func (p *parser) cur() token { return p.toks[p.at] }
func (p *parser) done() bool { return p.toks[p.at].kind == tokEOF }
func (p *parser) pos() Pos   { return Pos{File: p.name, Line: p.cur().line} }
func (p *parser) advance() {
	if !p.done() {
		p.at++
	}
}

func (p *parser) peekAt(n int) token {
	if p.at+n >= len(p.toks) {
		return p.toks[len(p.toks)-1]
	}
	return p.toks[p.at+n]
}

func (p *parser) isIdent(s string) bool {
	t := p.cur()
	return t.kind == tokIdent && t.text == s
}

func (p *parser) isPunct(s string) bool {
	t := p.cur()
	return t.kind == tokPunct && t.text == s
}

func (p *parser) accept(s string) bool {
	if p.isPunct(s) || p.isIdent(s) {
		p.advance()
		return true
	}
	return false
}

func (p *parser) refuse(pos Pos, kind, detail, reason string) {
	p.file.Refusals = append(p.file.Refusals, Refusal{
		Pos: pos, Kind: kind, Detail: truncate(detail), Reason: reason,
	})
}

func truncate(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}

// run walks the top level. Every branch either consumes a whole declaration or
// hands control to skipDeclaration, so the loop always makes progress.
func (p *parser) run() {
	for !p.done() {
		before := p.at
		p.topLevel()
		if p.at == before {
			p.advance()
		}
	}
}

func (p *parser) topLevel() {
	t := p.cur()
	switch {
	case t.kind == tokDirective:
		p.directive(t)
		p.advance()
	case t.kind == tokPunct:
		p.advance()
	case t.text == "native":
		p.native(nil)
	case t.text == "stock":
		p.stock()
	case t.text == "methodmap":
		p.methodmap()
	case t.text == "enum" && p.peekAt(1).text == "struct":
		p.enumStruct()
	case t.text == "enum":
		p.enum()
	case t.text == "typedef":
		p.typedef()
	case t.text == "typeset":
		p.typeset()
	default:
		p.skipDeclaration()
	}
}

// skipDeclaration consumes one statement or braced block without interpreting
// it: stocks, forwards, typesets, globals and anything unrecognised.
func (p *parser) skipDeclaration() {
	depth := 0
	for !p.done() {
		switch {
		case p.isPunct("{") || p.isPunct("("):
			depth++
		case p.isPunct("}") || p.isPunct(")"):
			depth--
			if depth <= 0 {
				p.advance()
				if depth == 0 && p.isPunct(";") {
					p.advance()
				}
				return
			}
		case p.isPunct(";") && depth == 0:
			p.advance()
			return
		case p.cur().kind == tokDirective:
			if depth == 0 {
				return
			}
		}
		p.advance()
	}
}

func (p *parser) directive(t token) {
	rest, ok := strings.CutPrefix(t.text, "define")
	if !ok || (rest != "" && !isSpaceByte(rest[0])) {
		return
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return
	}
	name := fields[0]
	if strings.ContainsAny(name, "(") {
		p.refuse(Pos{p.name, t.line}, "define", t.text, "function-like macro")
		return
	}
	p.file.Defines = append(p.file.Defines, Define{
		Name:  name,
		Value: strings.TrimSpace(rest[strings.Index(rest, name)+len(name):]),
		Pos:   Pos{p.name, t.line},
	})
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' }

// native parses `native <type> <name>(params);`. When owner is non-nil the
// declaration is a methodmap member and the result is appended there instead.
func (p *parser) native(owner *Methodmap) {
	start := p.cur()
	p.advance() // "native"
	ret, ok := p.parseReturnType()
	if !ok {
		p.refuse(Pos{p.name, start.line}, "native", p.textFrom(start.line), "unreadable return type")
		p.skipDeclaration()
		return
	}
	name := ret.Name
	switch {
	case p.cur().kind == tokIdent:
		name = p.cur().text
		p.advance()
	case owner == nil:
		p.refuse(Pos{p.name, start.line}, "native", p.textFrom(start.line), "missing name")
		p.skipDeclaration()
		return
	default:
		// A methodmap member with no name after its type is a constructor.
		ret = Type{Name: owner.Name}
	}
	params, ok := p.parseParams()
	if !ok {
		p.refuse(Pos{p.name, start.line}, "native", p.textFrom(start.line), "unreadable parameter list")
		p.skipDeclaration()
		return
	}
	p.accept(";")
	if owner == nil {
		p.file.Natives = append(p.file.Natives, Native{
			Name: name, Return: ret, Params: params,
			Doc: start.doc, Pos: Pos{p.name, start.line},
		})
		return
	}
	kind := MethodPlain
	if name == owner.Name {
		kind = MethodConstructor
	}
	owner.Methods = append(owner.Methods, Method{
		Name: name, Kind: kind, Return: ret, Params: params, Native: true,
		Doc: start.doc, Pos: Pos{p.name, start.line},
	})
}

// stock parses `stock <type> <name>(params) { body }`. The body is skipped:
// a stock is called exactly like a native, and only its signature is a
// binding.
func (p *parser) stock() {
	start := p.cur()
	p.advance() // "stock"
	p.accept("static")
	ret, ok := p.parseReturnType()
	if !ok || p.cur().kind != tokIdent {
		p.refuse(Pos{p.name, start.line}, "stock", p.textFrom(start.line), "unreadable signature")
		p.skipDeclaration()
		return
	}
	name := p.cur().text
	p.advance()
	if name == "operator" {
		p.refuse(Pos{p.name, start.line}, "stock", p.textFrom(start.line),
			"operator overload has no Go equivalent")
		p.skipDeclaration()
		return
	}
	params, ok := p.parseParams()
	if !ok {
		p.refuse(Pos{p.name, start.line}, "stock", p.textFrom(start.line), "unreadable parameter list")
		p.skipDeclaration()
		return
	}
	p.skipBody()
	p.file.Stocks = append(p.file.Stocks, Native{
		Stock: true, Name: name, Return: ret, Params: params,
		Doc: start.doc, Pos: Pos{p.name, start.line},
	})
}

// textFrom rebuilds the source text of the declaration starting at line, for
// refusal messages. It is only ever used on the error path.
func (p *parser) textFrom(line int32) string {
	var b strings.Builder
	for i := p.at; i < len(p.toks) && b.Len() < 200; i++ {
		if p.toks[i].line > line+4 {
			break
		}
		b.WriteString(p.toks[i].text)
		b.WriteByte(' ')
	}
	return b.String()
}
