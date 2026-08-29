// Package spshell runs a pure SourcePawn function under SourcePawn's standalone
// VM, so a generated function can be compared with the Go it came from without a
// game server.
//
// # The toolchain
//
// spshell is not shipped in the SourceMod drop the testbed downloads; only
// spcomp is. It has to be built from alliedmodders/sourcepawn:
//
//	python3 -m venv venv
//	./venv/bin/pip install git+https://github.com/alliedmodders/ambuild
//	git clone --recursive https://github.com/alliedmodders/sourcepawn
//	cd sourcepawn && mkdir objdir && cd objdir
//	../../venv/bin/python ../configure.py --enable-optimize
//	../../venv/bin/ambuild
//
// That produces objdir/spshell/linux-x86_64/spshell and, next to it,
// objdir/spcomp/linux-x86_64/spcomp.
//
// Compiling with that spcomp is not optional. The SourceMod 1.12 spcomp64 in
// testbed/build implicitly includes its own float.inc, which declares the float
// operators as __FLOAT_DIV__ and friends; spshell binds them lowercase, as
// __float_div, so an SM-compiled plugin dies on the first division with
// "Native is not bound: __FLOAT_DIV__". The differential test therefore compiles
// with the sourcepawn spcomp and its include tree, and it is the plugin build
// that keeps using the SourceMod one.
//
// Point the tests at the toolchain with SPSHELL, SPCOMP and SPINCLUDE; without
// them the tests skip.
package spshell
