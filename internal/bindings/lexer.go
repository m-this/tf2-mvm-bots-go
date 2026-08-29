package bindings

import "strings"

type tokKind uint8

const (
	tokEOF tokKind = iota
	tokIdent
	tokNumber
	tokString
	tokChar
	tokPunct
	tokDirective // a whole `#...` line, Text is the line without the leading '#'
)

type token struct {
	kind tokKind
	text string
	line int32
	doc  string // block or line comment immediately above, already trimmed
}

// maxTokens bounds the lexer so a malformed file cannot spin forever. The
// largest include in the tree is under 60k tokens.
const maxTokens = 1 << 21

type lexer struct {
	src  []byte
	pos  int
	line int32

	doc     string
	docLine int32
}

func lex(src []byte) []token {
	l := &lexer{src: src, line: 1}
	toks := make([]token, 0, len(src)/6+1)
	for len(toks) < maxTokens {
		t, ok := l.next()
		if !ok {
			break
		}
		toks = append(toks, t)
	}
	return append(toks, token{kind: tokEOF, line: l.line})
}

func (l *lexer) next() (token, bool) {
	atLineStart := l.skipSpaceAndComments()
	if l.pos >= len(l.src) {
		return token{}, false
	}
	c := l.src[l.pos]
	switch {
	case c == '#' && atLineStart:
		return l.emit(tokDirective, l.directive()), true
	case isIdentStart(c):
		return l.emit(tokIdent, l.span(isIdentPart)), true
	case c >= '0' && c <= '9':
		return l.emit(tokNumber, l.span(isNumberPart)), true
	case c == '"':
		return l.emit(tokString, l.quoted('"')), true
	case c == '\'':
		return l.emit(tokChar, l.quoted('\'')), true
	default:
		return l.emit(tokPunct, l.punct()), true
	}
}

func (l *lexer) emit(k tokKind, text string) token {
	t := token{kind: k, text: text, line: l.line}
	if l.doc != "" && l.docLine >= l.line-1 {
		t.doc = l.doc
	}
	l.doc = ""
	return t
}

// skipSpaceAndComments advances past whitespace and comments, remembering the
// last comment as a candidate doc comment. It reports whether the next
// character is the first non-space character of its line.
func (l *lexer) skipSpaceAndComments() bool {
	atLineStart := l.pos == 0
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		switch {
		case c == '\n':
			l.line++
			l.pos++
			atLineStart = true
		case c == ' ' || c == '\t' || c == '\r' || c == '\f' || c == '\v':
			l.pos++
		case c == '\\' && l.peekNext() == '\n':
			l.line++
			l.pos += 2
		case c == '/' && l.peekNext() == '/':
			l.lineComment()
			atLineStart = true
		case c == '/' && l.peekNext() == '*':
			l.blockComment()
		default:
			return atLineStart
		}
	}
	return atLineStart
}

// peekNext returns the character after the current one, or 0 at the end.
func (l *lexer) peekNext() byte {
	if l.pos+1 >= len(l.src) {
		return 0
	}
	return l.src[l.pos+1]
}

func (l *lexer) lineComment() {
	start := l.pos + 2
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.pos++
	}
	text := strings.TrimSpace(string(l.src[start:l.pos]))
	if l.docLine == l.line-1 && l.doc != "" {
		l.doc += "\n" + text
	} else {
		l.doc = text
	}
	l.docLine = l.line
}

func (l *lexer) blockComment() {
	start := l.pos + 2
	l.pos += 2
	for l.pos < len(l.src) {
		if l.src[l.pos] == '\n' {
			l.line++
		}
		if l.src[l.pos] == '*' && l.peekNext() == '/' {
			break
		}
		l.pos++
	}
	end := min(l.pos, len(l.src))
	l.pos = min(l.pos+2, len(l.src))
	l.doc = cleanBlockComment(string(l.src[start:end]))
	l.docLine = l.line
}

func cleanBlockComment(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		ln = strings.TrimPrefix(ln, "*")
		out = append(out, strings.TrimSpace(ln))
	}
	for len(out) > 0 && out[0] == "" {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// directive consumes a whole preprocessor line, honouring backslash
// continuations, and returns it without the leading '#'.
func (l *lexer) directive() string {
	l.pos++ // '#'
	var b strings.Builder
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\\' && l.peekNext() == '\n' {
			b.WriteByte(' ')
			l.line++
			l.pos += 2
			continue
		}
		if c == '\n' {
			break
		}
		if c == '/' && (l.peekNext() == '/' || l.peekNext() == '*') {
			break
		}
		b.WriteByte(c)
		l.pos++
	}
	return strings.TrimSpace(b.String())
}

func (l *lexer) span(pred func(byte) bool) string {
	start := l.pos
	for l.pos < len(l.src) && pred(l.src[l.pos]) {
		l.pos++
	}
	return string(l.src[start:l.pos])
}

func (l *lexer) quoted(q byte) string {
	start := l.pos
	l.pos++
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\\' {
			l.pos += 2
			continue
		}
		if c == '\n' {
			break
		}
		l.pos++
		if c == q {
			break
		}
	}
	return string(l.src[start:min(l.pos, len(l.src))])
}

var multiPunct = []string{"...", "<<=", ">>=", "==", "!=", "<=", ">=", "&&", "||", "<<", ">>", "++", "--", "+=", "-=", "*=", "/=", "|=", "&=", "^=", "::"}

func (l *lexer) punct() string {
	rest := l.src[l.pos:]
	for _, p := range multiPunct {
		if len(rest) >= len(p) && string(rest[:len(p)]) == p {
			l.pos += len(p)
			return p
		}
	}
	l.pos++
	return string(rest[:1])
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '@' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func isNumberPart(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') ||
		c == 'x' || c == 'X' || c == '.' || c == '_'
}
