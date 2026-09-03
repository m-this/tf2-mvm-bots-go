/*
Package slots is how many client slots the mod keeps state for.

One number, and it was written 41 times: every package holding a per-client
array declared its own const Slots = 65, because a body could import nothing but
the engine. Each of those emitted a #define of its own into SourcePawn's one
flat namespace, and spcomp warned about the redefinition on 39 of them, every
build, for as long as the port has existed.

It is a constant and not a call, so a caller folds it: the value is written
where it is used and this package's own declaration is the only #define. That
also makes the array length a constant expression, which is what SourcePawn
needs to size an array at all.
*/
package slots

// Count is MAXPLAYERS + 1, so a client index is its own subscript and slot zero
// stays empty the way the game leaves it.
const Count = 65
