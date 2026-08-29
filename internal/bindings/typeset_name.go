package bindings

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

/*
	Naming the call signatures of a typeset

Sig1 to Sig29 numbered them by their position in the include, so a signature
inserted upstream renamed every one after it. That is the features.sp bug this
repository opened with: position carrying meaning a name should carry.

Two sources of a stable name, in this order.

The comment above a variant, where it is a list of the callbacks that signature
serves and nothing else: ActionHandlerOnUpdate reads as what it is. 146 of the
280 variants have one, and the actions headers are where the callbacks people
write live, so this covers the names that get typed.

Otherwise the signature itself, tagged with four hex digits of its rendered
parameters and result: ActionEventResponderCallbackSig7a3c. Two variants of one
typeset cannot render the same, or they would be the same variant, so the tag is
unique by construction and moves only when that signature's own types move.
*/

// callbackList matches a doc line that is only a comma-separated list of
// identifiers. Anything else is prose: "@return True to allow the current
// entity to be hit" is a sentence about the signature, not a name for it.
var callbackList = regexp.MustCompile(`^[A-Za-z_]\w*(, *[A-Za-z_]\w*)*$`)

// signatureTag is four hex digits of the rendered signature, which is what
// makes the name independent of where the variant sits in the include.
func signatureTag(params, result string) string {
	h := fnv.New32a()
	// Write on a hash never returns an error, and errcheck is right to ask.
	_, _ = h.Write([]byte(params + "|" + result))
	return fmt.Sprintf("%04x", h.Sum32()&0xffff)
}

// docName reads the callbacks a variant serves off its comment, or returns "".
func docName(doc string) string {
	line, _, _ := strings.Cut(doc, "\n")
	line = strings.TrimSpace(line)
	if line == "" || !callbackList.MatchString(line) {
		return ""
	}
	first, _, _ := strings.Cut(line, ",")
	return strings.TrimSpace(first)
}

/*
	variantNames names every call signature of one typeset

Done for the whole typeset at once, because a doc name two variants share tells
us nothing about either: both fall back to their tag rather than one winning by
being first, which would put position back into the name through the side door.

renders is one "params|result" string per variant, in variant order. A variant
whose signature could not be rendered gets "" and is named by tag, so a refusal
later does not shift the names of the ones that did render.
*/
func variantNames(owner string, variants []TypesetVariant, renders []string) []string {
	wanted := make([]string, len(variants))
	seen := map[string]int{}
	for i, v := range variants {
		if name := docName(v.Doc); name != "" {
			wanted[i] = name
			seen[name]++
		}
	}

	names := make([]string, len(variants))
	for i := range variants {
		if wanted[i] != "" && seen[wanted[i]] == 1 {
			names[i] = owner + goIdent(wanted[i])
			continue
		}
		params, result, _ := strings.Cut(renders[i], "|")
		names[i] = owner + "Sig" + signatureTag(params, result)
	}
	return names
}
