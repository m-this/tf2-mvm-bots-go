/*
Package sp is what SourcePawn's own syntax requires of the text we generate.

Small on purpose. It holds the lexical facts that more than one generator has to
get right, so getting one wrong is one fix rather than a hunt: internal/spshell
writes golden inputs and internal/spgen writes constants, and a float that reads
back as a different float in either is the same silent wrong answer.
*/
package sp

import (
	"strconv"
	"strings"
)

/*
	FloatLiteral writes v in the form spcomp's lexer takes

Shortest decimal that reads back as the same float32. Plain decimal for anything
Go writes that way, exponent form for the rest, because 'f' form is where this
went wrong: FLT_MAX as 39 digits compiled to 0x5f794ad1, about 1.8e19, and
spcomp said nothing.

Two rules for the exponent form, both found by compiling. The mantissa needs a
point with a digit each side, so 1e-45 is "number literal has invalid digits"
and 1.0e-45 is the smallest denormal. And a positive exponent carries no sign,
so 3.4028235e+38 is "exponential must be followed by integer".

Fuzzed against spcomp in internal/spshell. See mvm-z83.15.
*/
func FloatLiteral(v float32) string {
	s := strconv.FormatFloat(float64(v), 'g', -1, 32)
	mantissa, exponent, hasExponent := strings.Cut(s, "e")
	if !strings.ContainsRune(mantissa, '.') {
		mantissa += ".0"
	}
	if !hasExponent {
		return mantissa
	}
	return mantissa + "e" + strings.TrimPrefix(exponent, "+")
}
