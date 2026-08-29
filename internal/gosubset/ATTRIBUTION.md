# SourceGo

The rule set in this package descends from SourceGo / Go2SourcePawn,
<https://github.com/Nirari-Technologies/Go2SourcePawn>, MIT licensed,
Copyright (c) 2020 Kevin Yonan. Read, not forked; `docs/design.md` says why.

    MIT License

    Copyright (c) 2020 Kevin Yonan

    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.

## Taken from `rewrite/srcgo/pass1_illegal_code.go`

The list of constructs that have no SourcePawn, and it is the same list:
`goto`, `fallthrough`, labels and labelled branches, `defer`, `go`, `select`,
channel types, sends and receives, type switches, type assertions, slice
expressions, imaginary literals, pointer results, pointers in struct fields,
and arrays of unknown size in struct fields. Every one of those is refused here
for the reason pass1 refuses it.

## Taken from `rewrite/srcgo/pass3_merge_rettypes.go`

Several return values are accepted rather than refused, because the generator
turns the ones after the first into by-reference parameters, exactly as pass3
does. This is why `*T` is refused everywhere in author-written code: the only
pointers in the output are the ones the generator puts there.

## Taken from `rewrite/srcgo/pass9_mutate_ranges.go`

`for range` is accepted and lowered to an indexed loop, so the checker does not
refuse it. pass9's rewrite is the reason it can be accepted: a range key becomes
a generated index name and a range value becomes an indexed read at the top of
the body.

## Taken from `rewrite/srcgo/pass10_mutate_no_ret_calls.go`

A call statement whose result is discarded is accepted, because pass10 declares
a temporary for each dropped result and passes it by reference. Without that,
`g(a)` for a `g` that returns something would have to be refused.

## Not taken

- **String-keyed maps.** pass1 permits `map[string]V` and refuses only non-string
  maps, because SourceGo maps them onto `StringMap`. This subset has no strings
  and no maps: an attribute name is an `int32` identifier, and the name table
  lives in `internal/tables` as a single source of truth. `GeneralUpgradePriority`
  is a chain of forty `StrEqual` calls today; in Go it is a `switch` over an
  `int32`.
- **The SourcePawn front end.** pass1 runs against SourceGo's own tokenizer,
  preprocessor, two parsers and typechecker in `rewrite/sptools/`. `spcomp` is
  the front end whose opinion counts here, so none of that is carried.
- **Function pointers.** pass10 has a whole branch for calls through function
  pointers, expanded into a chain of calls. The subset has no function values,
  so the branch has nothing to check.
- **Silence.** SourceGo's `PrintErr` names the construct. This package also
  names what to write instead, in every refusal, because the reader is somebody
  who just found out their function is not translatable and needs the next move.
- **Permissiveness about what a body may reach.** pass1 checks constructs and
  says nothing about which functions may be called. Here a call must resolve to
  a function in the same package, a declared native binding, or a mapped
  identifier of a mapped import; anything else is refused by name.
