#!/bin/sh
# Turn a core file into a backtrace with names in it.
#
#   testbed/symbolise-core.sh core.1234
#
# An unsymbolised trace kept one watchdog crash open for two sessions: every
# frame reads as "?? ()" because the shipped binaries are stripped of everything
# except the dynamic symbols, and gdb will not find the libraries on its own
# because they live under the server's own tree rather than the system's.
#
# So the sysroot is the server tree and the search path names every directory
# the game loads from. With those, the same core names CNavArea::GetZ under
# NavAreaBuildPath in one go.
set -eu

core=${1:?usage: symbolise-core.sh <core file>}

: "${TESTBED_NATIVE_ROOT:=$HOME/tf2-native/tf-dedicated}"

root=$TESTBED_NATIVE_ROOT

command -v gdb >/dev/null || { echo "gdb is not installed"; exit 1; }

exec gdb -batch \
	-ex "set sysroot $root" \
	-ex "set solib-search-path $root/bin:$root/tf/bin:$root/tf/addons/sourcemod/bin:$root/tf/addons/sourcemod/extensions:$root/tf/addons/metamod/bin" \
	-ex "bt 30" \
	"$root/srcds_linux" "$core" 2>&1 | grep -vE "^warning|^\[New LWP|^$"
