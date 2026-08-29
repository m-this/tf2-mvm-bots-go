/*
Package spbody translates a Go package that passes internal/gosubset into the
SourcePawn the plugin includes.

It is the piece mvm-bis is about. internal/spgen emits tables by running the Go
and recording what it answered, which cannot disagree with the Go it came from
but only works for a decision that reads plain values. A function that calls the
engine cannot be run here at all, so its SourcePawn has to be translated from the
syntax, and this is that translation.

The Go is type checked with go/types before anything is emitted, so an integer
width, a named type, a struct field and an array length are read off the type
rather than guessed from the syntax. A body that does not type check is not
translated: the Go compiler is the first gate and this is the second.

# What a body may reach outside itself

Nothing, except the externs. An extern is a function the body calls and this
package does not translate, because the engine already has it: a native, an
SDKCall handle prepared at load, or a raw address read. Externs are declared in
one generated file per body package, which carries both the Go declaration the
body compiles against and the SourcePawn call form emitted at each call site.
That file is generated, so it is not authored Go and neither gosubset nor this
package translates it; Generated reports it and skips it.

The Go declaration of an extern dispatches through a package-level Natives
value. Production has none and every call panics, which is correct: a body that
calls the engine has no meaning in a Go process. The differential test installs
one, which is how a body holding an engine call keeps the proof the pure ones
have: the same stub answers on both sides, and the two traces have to match.
*/
package spbody
