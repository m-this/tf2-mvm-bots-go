#!/bin/sh
# Put a copy of the game where the native test-bed can run it.
#
#   testbed/seed-native.sh                  from the container test-bed's volume
#   TESTBED_SEED_FROM=<container> testbed/seed-native.sh
#
# A copy and never a share, for the same reason the container bed copies rather
# than mounting: this installs plugins over addons/, and doing that to a tree a
# live server is reading is how a day was once lost to SIGBUS.
#
# Core files are left behind: they are the reason this bed exists but they are
# hundreds of megabytes each and the copy is large enough already.
set -eu

: "${TESTBED_NATIVE_ROOT:=$HOME/tf2-native/tf-dedicated}"
: "${TESTBED_SEED_FROM:=mvmbots-testbed-srcds-1}"

dest=$(dirname "$TESTBED_NATIVE_ROOT")

docker inspect "$TESTBED_SEED_FROM" >/dev/null 2>&1 || {
	echo "no container called $TESTBED_SEED_FROM to copy from"
	echo "bring the container bed up once with testbed/run.sh, or name another with TESTBED_SEED_FROM"
	exit 1
}

mkdir -p "$dest"

echo "[native] copying the game out of $TESTBED_SEED_FROM, about fifteen gigabytes"

# Anchored at the tree root on purpose. An unanchored core.* also matches
# SourceMod's own core.games.txt and core.phrases.txt, and a server missing those
# starts, fails to parse its gamedata, and never answers rcon.
docker exec "$TESTBED_SEED_FROM" tar cf - --exclude="tf-dedicated/core.*" --exclude="*.jsonl" \
	-C /home/steam tf-dedicated | tar xf - -C "$dest"

# The server needs a 32 bit C++ runtime and the host may not have one. srcds_run
# puts bin/ on the library path, so a copy of the image's own lives there and the
# tree stays self contained rather than asking for a package to be installed.
if [ ! -e "$TESTBED_NATIVE_ROOT/bin/libstdc++.so.6" ]; then
	echo "[native] taking the 32 bit C++ runtime from the image"
	docker cp "$TESTBED_SEED_FROM:/usr/lib32/libstdc++.so.6.0.30" "$TESTBED_NATIVE_ROOT/bin/libstdc++.so.6" 2>/dev/null || \
		echo "[native] could not copy it; install libstdc++6:i386 if the server will not start"
fi

echo "[native] seeded $TESTBED_NATIVE_ROOT"
