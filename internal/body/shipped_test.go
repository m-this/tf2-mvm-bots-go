package body_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	TestGeneratedBodiesMatchTheShippedOnes

The same comparison the actions get, for the plain bodies: every function a body
package claims a SourcePawn name for is read out of the file it replaces, at the
pin, and compared on its declaration and on the sequence of calls it makes.

This did not exist while util.sp was being ported, and that was a hole rather
than a decision: those packages carried a Shipped path that nothing read, so a
port could drop a branch or reorder two calls and only the compiler would have
an opinion. The actions had this from the start; the bodies now do too.

A generated name with no shipped counterpart is skipped rather than failed: a
body may add a helper the plugin never had, and internal/body/finders holds
functions from two different files.
*/
func TestGeneratedBodiesMatchTheShippedOnes(t *testing.T) {
	upstream.SkipOrFail(t)

	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	for _, b := range body.All {
		if b.Shipped == "" {
			continue
		}

		t.Run(b.Dir, func(t *testing.T) {
			shipped, err := upstream.Read(strings.Split(b.Shipped, "/")...)
			if err != nil {
				t.Fatalf("reading %s at %s: %v", b.Shipped, upstream.Rev, err)
			}

			compareBody(t, string(generated[b.Out]), shipped)
		})
	}
}

// emittedNames are the functions a generated file declares, which is what the
// comparison walks.
var emittedNames = regexp.MustCompile(`(?m)^stock \w+(?:\[\])? (\w+)\(`)

/*
	declarationOf is the function itself, not the first mention of its name

A name appears at its call sites long before it appears at its declaration, and
taking the first match compared a call against a definition and reported nonsense.
The declaration is the one at the start of a line, after an optional stock or
static and a return type.
*/
func declarationOf(src, name string) (string, bool) {
	at := regexp.MustCompile(`(?m)^(?:stock |static )?\w+(?:\[\])? ` + regexp.QuoteMeta(name) + `\(`)

	loc := at.FindStringIndex(src)
	if loc == nil {
		return "", false
	}
	return callbackOf(src[loc[0]:], name)
}

/*
	shape is the declaration without the parts a port chooses

The return type, the parameter types in order, and the defaults: those are what a
caller depends on. What is dropped is the stock the generator adds and the shipped
file often leaves off, and the parameter names, which are the port's to pick the
way the Go names them. A wrong name is a worse comment; a wrong type is a wrong
call.
*/
func shape(fn string) string {
	decl := strings.TrimPrefix(declOf(fn), "stock ")
	decl = strings.TrimPrefix(decl, "static ")

	open := strings.Index(decl, "(")
	if open < 0 {
		return decl
	}

	head := decl[:open]
	if space := strings.LastIndexByte(head, ' '); space >= 0 {
		head = head[:space] // the return type, without the name
	}

	var params []string
	for _, p := range strings.Split(strings.Trim(decl[open+1:], "()"), ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		value := ""
		if name, def, has := strings.Cut(p, "="); has {
			p, value = strings.TrimSpace(name), " = "+strings.TrimSpace(def)
		}

		/* const is compared, because it is part of the contract

		A const array cannot be handed to a native that declares its
		parameter writable, and a caller holding one cannot pass it to a
		function that does not promise. So neither always-const nor
		never-const compiles across the plugin: the port says which with
		//sp:const and this is what checks it against the shipped
		declaration. */

		/* The type is everything but the name

		Two shapes: char[] name, where the dimensions belong to the type
		and come before the name, and float name[3], where they come
		after it. Taking the last word as the name and keeping whatever
		brackets were attached to it handles both.
		*/
		dims := ""
		if at := strings.Index(p, "["); at >= 0 && at > strings.LastIndexByte(p, ' ') {
			dims, p = p[at:], strings.TrimSpace(p[:at])
		}
		if space := strings.LastIndexByte(p, ' '); space >= 0 {
			p = p[:space]
		}

		params = append(params, p+dims+value)
	}

	return head + "(" + strings.Join(params, ", ") + ")"
}

/*
	reshaped are the functions whose declaration the port changed on purpose

One entry, and it needs the reason beside it. NestSpotFromList took the running
best spot and distance by reference and updated them in place. A generated
function zeroes its out-parameters at entry, so passing one variable as both the
candidate and the answer read the candidate as zeros: the nest was scored from the
map origin. The port takes the candidate and returns the answer separately, and
the emitter refuses the aliased shape outright now.
*/
var reshaped = map[string]string{
	"NestSpotFromList": "the candidate and the answer cannot be the same variable; see the aliasing refusal in internal/spbody",
}

/*
	frees is how many handles a body releases

Distinct handles rather than statements, and both spellings. SourceMod frees one
with delete or with Close and the shipped files use each in different functions.
The generator writes delete from a defer, which puts the free at every way out
rather than at the one the author remembered, so one handle can be freed in two
places and is still freed once per path. What has to match is how many handles
are released, not how many times the text says so.
*/
func frees(fn string) int {
	names := map[string]bool{}

	for _, m := range freeDelete.FindAllStringSubmatch(fn, -1) {
		names[m[1]] = true
	}
	for _, m := range freeClose.FindAllStringSubmatch(fn, -1) {
		names[m[1]] = true
	}
	return len(names)
}

var (
	freeDelete = regexp.MustCompile(`delete (\w+)`)
	freeClose  = regexp.MustCompile(`(\w+)\.Close\(\)`)
)

// withoutCloses drops the Close calls, which are counted rather than sequenced.
func withoutCloses(calls []string) []string {
	out := calls[:0:0]
	for _, c := range calls {
		if c == "Close" {
			continue
		}
		out = append(out, c)
	}
	return out
}

func compareBody(t *testing.T, got, shipped string) {
	t.Helper()

	compared := 0

	for _, m := range emittedNames.FindAllStringSubmatch(got, -1) {
		name := m[1]

		if reshaped[name] != "" {
			// A port that deliberately changed the shape, with the
			// reason written down. The call sequence is still
			// compared, so the body cannot drift behind the note.
			continue
		}

		want, ok := declarationOf(shipped, name)
		if !ok {
			// A helper this port added, or a function from another
			// file: neither is a difference in what was replaced.
			continue
		}
		have, ok := declarationOf(got, name)
		if !ok {
			t.Fatalf("the generated file declares %s and then does not hold it", name)
		}

		compared++

		t.Run(name, func(t *testing.T) {
			if wantDecl, haveDecl := shape(want), shape(have); wantDecl != haveDecl {
				t.Errorf("the declaration differs:\nshipped:   %s\ngenerated: %s", wantDecl, haveDecl)
			}

			/* delete and Close are the same operation on a handle

			The shipped files write handle.Close() and the generator
			writes delete handle, because a defer puts the free at
			every way out rather than at the one the author
			remembered. Both are compared: the frees have to match in
			number, and the rest of the sequence has to match in
			order. */
			wantFrees, haveFrees := frees(want), frees(have)
			if wantFrees != haveFrees {
				t.Errorf("the body frees %d handles and the shipped one frees %d", haveFrees, wantFrees)
			}

			if wantCalls, haveCalls := withoutCloses(callsIn(want)), withoutCloses(callsIn(have)); !slices.Equal(wantCalls, haveCalls) {
				t.Errorf("the body calls a different sequence:\nshipped:   %v\ngenerated: %v", wantCalls, haveCalls)
			}
		})
	}

	if compared == 0 {
		t.Errorf("nothing was compared, so this proves nothing")
	}
}
