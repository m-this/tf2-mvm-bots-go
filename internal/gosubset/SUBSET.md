# The Go subset the body generator translates

One page. Anything not on it is refused with `file:line:col`, the construct, and
what to write instead. Refusing is the point: a generator that half supports a
construct produces a plugin that compiles and is wrong.

## What a body is

A generated body is one function, called by hand-written SourcePawn, taking
plain values and returning plain values. It reaches the engine only through the
extern package, `internal/engine`. It allocates nothing.

It may own state: a package-level `var` is the plugin's own global, emitted as a
SourcePawn global with the same initial value. What SourcePawn will not do is
compute one at load, so the initialiser is a constant or an array of constants
and anything else is refused. Putting the state back between maps is the body's
own job, written as an ordinary function so the differential test walks it.

## Types

Accepted:

- `bool`
- `int8 int16 int32 int64 uint8 uint16 uint32 uint64` and the aliases `byte`,
  `rune`
- `float32`, `float64`
- fixed-length arrays, `[N]T`, where `N` is a literal, a named constant or
  arithmetic over them
- named structs declared at package level, whose fields are any of the above
- named types over any of the above, which is how an enum is written:
  `type ThreatPriority int32` plus a `const` block with `iota`

Refused, with the fix:

| Written | Refused because | Write instead |
| --- | --- | --- |
| `int`, `uint` | a cell is 32 bits and the source should say so | `int32`, `uint32` |
| `string` | no strings | an `int32` identifier, with the name table in `internal/tables` |
| `[]T` | nothing grows | `[N]T` plus a count parameter |
| `map[K]V` | no maps | `[N]T` indexed by a small `int32` |
| `*T` | no pointers | pass the value; several results become by-reference parameters in the generated function |
| `interface`, `any`, `error` | no dispatch on type | the concrete type, and a `bool` or sentinel `int32` for failure |
| `chan T`, `func(...)` | no concurrency, no function values | remove it, call the function by name |
| anonymous struct, embedded field, type alias, generics | nothing to name in the output | a named package-level struct with named fields, one function per concrete type |

## Declarations

- Functions, with parameters and results of accepted types. Several results are
  fine: the generator turns the ones after the first into by-reference
  parameters, and every result that becomes a parameter has to be named,
  because the name is what the parameter is called.
- A result that is an array, which SourcePawn cannot return: it becomes a
  trailing parameter and the function returns nothing. `v := Centre(s)` is
  therefore a declaration and a call, and a call returning an array used inside
  a larger expression is refused.
- `const`, including `iota` blocks.
- `//sp:default <parameter> <value>` on a function, which gives the emitted
  SourcePawn parameter a default. Go has none, so a Go caller passes everything
  and no behaviour depends on it: it is there so the plugin's existing call
  sites still compile while they are being ported, and it goes when they do.
- `type`, at package level only.
- Package-level `var`, whose initialiser is a constant or an array literal of
  constants. It becomes a SourcePawn global.
- `init`, methods, receivers and generic functions are refused. A methodmap is
  the actions generator's job, not this one.

## Statements

Accepted: `if` / `else`, three-clause `for`, condition-only `for`, `for {}`,
`for range` over an array or an integer, `switch` with a tag, `return`,
assignment including the compound operators, `++`/`--`, `break`, `continue`,
local `var` and `const`, and a call statement whose result is discarded.

Refused: `go`, `defer`, `select`, channel send and receive, labels, `goto`,
`fallthrough` (write `case a, b:`), type switch, and a `switch` with no tag
(SourcePawn's `switch` needs a value; write `if` / `else if`).

Switch cases must be constants, which is what SourcePawn accepts.

## Expressions

Accepted: identifiers, integer, float and character literals, arithmetic,
comparison, logical and bitwise operators, indexing, field access, composite
literals for arrays and structs, conversions to accepted types, `len`, `min`,
`max`, calls to functions declared in the same package, and calls to the
generated native bindings named in `Config.Natives`.

Refused: string and imaginary literals, `&x`, `*p`, slice expressions, type
assertions, function literals, `make`, `new`, `append`, `copy`, `clear`, `cap`,
`delete`, `panic`, `recover`, `print`, `println`, `complex`, `real`, `imag`,
and `&^` (SourcePawn has no AND NOT; write `x & ^y`).

## Imports

None, unless `Config.Packages` maps the path and the identifier. The default
maps a handful of `math` functions onto the SourcePawn float builtins. Since
`math` is `float64` and a SourcePawn float is 32-bit, a `float32` body writes
its own helper or calls a native binding instead.

The one import a real body has is `internal/engine`, which is how it calls the
engine: one Go function per call, carrying the directive that says whether
SourcePawn writes it as a native, an `SDKCall` or an address read.
`internal/body.SubsetConfig` reads those declarations and maps the package, so
a call the extern package does not declare is refused here rather than emitted
as something plausible.

## The unit of checking is the directory

`CheckDir` collects the package-level types and functions of every non-test
`.go` file in the directory before it checks any of them, so a package may be
split across as many files as it needs. A directory holding two package names
is an error rather than a merged name set. `CheckFiles` is the same thing over
files a caller has already parsed.

`CheckFile` and `CheckSource` check one file, and one file is all they know.
Package-level names come from that file alone, so a call to a function or a use
of a type declared in a sibling file is refused as unknown. That is correct for
a single self-contained body and wrong for a package: check a package with
`CheckDir`. Imports stay file-scoped either way, so one file importing `math`
does not let the next one write `math.Abs`.

## Why `range` is accepted without type information

The checker is syntactic. It is still sound for `range`, because the type rules
above leave nothing to range over but an array or an integer: there are no
slices, no maps, no channels and no function values anywhere in the subset, and
no import may introduce one. The same argument makes `len` safe.

## What the checker does not catch

- Integer width and overflow. `int32` arithmetic that overflows in Go overflows
  the same way in SourcePawn, but a `float64` body silently loses precision when
  it becomes a 32-bit SourcePawn float. Write `float32`.
- Division by zero, and array indices out of range. Both are runtime faults on
  either side; the differential test under `spshell` is what finds them.
- Recursion depth. SourcePawn has a small stack.
- Whether a named function actually exists with the arity used. The checker
  knows package-level names, not signatures; the Go compiler is the second gate
  and it runs anyway.
- Local names. An identifier in expression position is not looked up, so a
  misspelled variable is the Go compiler's to catch. Only call position and
  type position are checked against the collected names.
