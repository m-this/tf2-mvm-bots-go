# Generated SourcePawn

Written by `tf2-mvm-bots-go`, not by hand. Each file names the Go package it
came from in its first line.

Committed rather than generated at build time, because the plugin's build is a
shell script and a compiler and adding Go to it would put a second toolchain in
front of anybody who wants to build the mod. The cost of that choice is drift,
and `make check` pays it: a file here that is not what the generator writes
today fails the gate.

To refresh, from the repository root:

```sh
make adopt
```
