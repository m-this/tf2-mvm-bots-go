#!/bin/sh
# Build SourcePawn's standalone compiler and VM, so the differential test runs
# rather than skips.
#
# Cached the way testbed/build.sh caches spcomp: fetched once, rebuilt never.
# Delete $work to start over.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=${SPWORK:-$root/toolchain}
src="$work/sourcepawn"

# Pinned, and it has to be. The -Werror removal below is a patch against a tree
# that is clang only, and an unpinned patch rots: compiler/errors.cpp:241 has a
# signed/unsigned comparison GCC rejects and upstream never sees.
ref=f2d11ff1a80ac891b7fb7ba81ad78646d9232c3e

spcomp="$src/objdir/spcomp/linux-x86_64/spcomp"
spshell="$src/objdir/spshell/linux-x86_64/spshell"
if [ -x "$spcomp" ] && [ -x "$spshell" ]; then
	exit 0
fi

mkdir -p "$work"

if [ ! -d "$src" ]; then
	echo "fetching sourcepawn@$ref"
	git clone --quiet https://github.com/alliedmodders/sourcepawn "$src"
	git -C "$src" -c advice.detachedHead=false checkout --quiet "$ref"
	git -C "$src" submodule --quiet update --init --recursive
fi

# -Werror on a clang-only tree, deleted in place rather than carried as a patch
# file. One line, and the pin above is what keeps it honest.
sed -i "s/'-Werror',/'-Wno-error',/" "$src/AMBuildScript"

# AMBuild is not on PyPI, so it is a clone too, and pinned for the same reason.
amb="$work/ambuild"
amb_ref=9ed1920068c8a767ae78022102abc93a6822eaad
if [ ! -d "$amb" ]; then
	echo "fetching ambuild@$amb_ref"
	git clone --quiet https://github.com/alliedmodders/ambuild "$amb"
	git -C "$amb" -c advice.detachedHead=false checkout --quiet "$amb_ref"
fi

venv="$work/venv"
if [ ! -x "$venv/bin/ambuild" ]; then
	python3 -m venv "$venv"
	"$venv/bin/pip" install --quiet "$amb"
fi

mkdir -p "$src/objdir"
# The build is loud: dropping -Werror leaves every warning GCC has about a
# clang-only tree on stderr. It goes to a log, and the log is what to read when
# the last line below fails.
log="$work/build.log"
(cd "$src/objdir" && "$venv/bin/python" ../configure.py --enable-optimize) >"$log" 2>&1
(cd "$src/objdir" && "$venv/bin/ambuild") >>"$log" 2>&1

test -x "$spcomp" && test -x "$spshell"
