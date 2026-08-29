package bindings

import "bytes"

// Code returns src with everything that is not a token blanked to spaces, and
// every string and character literal emptied. Line breaks are kept, so an
// offset and a line number still mean the same thing they did.
//
// It exists because scanning SourcePawn for the names it uses cannot scan the
// raw text: a commented-out prototype, a printf format and a VScript snippet
// passed as a string all read as call sites and none of them is one. The
// lexer already knows where a comment and a literal end, so this is the same
// knowledge rather than a second copy of it.
func Code(src []byte) []byte {
	out := bytes.Repeat([]byte{' '}, len(src))
	for i, c := range src {
		if c == '\n' {
			out[i] = '\n'
		}
	}
	for _, t := range lex(src) {
		switch t.kind {
		case tokEOF, tokString, tokChar:
			continue
		case tokIdent, tokNumber, tokPunct, tokDirective:
			copy(out[t.off:t.end], src[t.off:t.end])
		}
	}
	return out
}
