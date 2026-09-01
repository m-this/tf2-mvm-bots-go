#!/bin/sh
# Stage the mod, its dependencies and the statistics plugin into one SourceMod
# tree.
#
# The difference from the same script in tf2-archipelago, which this came from,
# is the one that matters here: the mod is compiled from the working tree above
# this directory rather than fetched from a tag. The point of the test-bed is
# to measure the change you just made.
#
# Everything else is downloaded and cached in $work, so a second run recompiles
# the mod and nothing else.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$root/testbed/versions.env"

work=${TESTBED_WORK:-$root/testbed/build}
out=${TESTBED_OUT:-$work/package}
src="$work/src"

mkdir -p "$src" \
	"$out/addons/sourcemod/plugins" \
	"$out/addons/sourcemod/extensions" \
	"$out/addons/sourcemod/gamedata" \
	"$out/addons/sourcemod/configs" \
	"$out/addons/sourcemod/logs"

git config --global advice.detachedHead false 2>/dev/null || true

# $1 repo, $2 ref, $3 directory
fetch() {
	[ -d "$src/$3" ] && return 0
	echo "fetching $1@$2"
	git clone --quiet --depth 1 --branch "$2" "https://github.com/$1" "$src/$3"
}

# --- Sources the mod compiles against ---

fetch OfficerSpy/SM_Stock_OfficerSpy "$SM_STOCK_OFFICERSPY_REF" stocklib
fetch FlaminSarge/tf2attributes "$TF2ATTRIBUTES_VERSION" tf2attributes
fetch nosoop/SM-TFEconData "$TFECONDATA_VERSION" tf_econ_data
fetch nosoop/SM-TFUtils "$TF2UTILS_VERSION" tf2utils
fetch nosoop/stocksoup "$STOCKSOUP_REF" stocksoup-root/stocksoup
fetch TF2-DMB/CBaseNPC "$CBASENPC_VERSION" cbasenpc
fetch Vinillia/actions.ext "$ACTIONS_VERSION" actions

# TF2Attributes does not compile as it ships: a #pragma unused sits above the
# declaration of UnloadAttributeValue rather than below it, and spcomp treats
# that as an error. One line, deleted in place rather than carried as a patch
# file, because a test-bed is not where a patch stack belongs.
sed -i '/^#pragma unused UnloadAttributeValue$/d' "$src/tf2attributes/scripting/tf2attributes.sp"

# --- The compiler ---

if [ ! -d "$work/spcomp" ]; then
	echo "fetching SourceMod $SOURCEMOD_VERSION"
	mkdir -p "$work/spcomp"
	curl -fsSL "https://sm.alliedmods.net/smdrop/$SOURCEMOD_BRANCH/sourcemod-$SOURCEMOD_VERSION-linux.tar.gz" |
		tar xz -C "$work/spcomp"
fi
sm="$work/spcomp/addons/sourcemod/scripting"

if [ ! -d "$work/ripext" ]; then
	echo "fetching ripext $RIPEXT_VERSION"
	mkdir -p "$work/ripext"
	curl -fsSL -o "$work/ripext.zip" \
		"https://github.com/ErikMinekus/sm-ripext/releases/download/$RIPEXT_VERSION/sm-ripext-$RIPEXT_VERSION-linux.zip"
	unzip -oq "$work/ripext.zip" -d "$work/ripext"
	rm -f "$work/ripext.zip"
fi

# -p, and it matters as much as the -u in the container's installer. These are
# the compiled extensions, and the running server has them mapped. A copy that
# gives them a new timestamp makes that installer copy them over the top on its
# next pass, which truncates a mapped file and kills the server in whichever
# extension was rewritten. The prebuilt tree is fetched once and never changes,
# so preserving its timestamps means an unchanged extension stays unchanged.
cp -rp "$work/ripext/addons/sourcemod/extensions/." "$out/addons/sourcemod/extensions/"

# --- The two compiled extensions ---
#
# Taken prebuilt. Building them needs a C++ toolchain and several CPU-minutes,
# and nothing this repository changes is inside either of them.

if [ ! -d "$work/prebuilt" ]; then
	mkdir -p "$work/prebuilt/cbasenpc" "$work/prebuilt/actions"

	echo "fetching CBaseNPC $CBASENPC_VERSION"
	curl -fsSL "https://github.com/TF2-DMB/CBaseNPC/releases/download/$CBASENPC_VERSION/cbasenpc${CBASENPC_VERSION}_linux.tar.gz" |
		tar xz -C "$work/prebuilt/cbasenpc"

	echo "fetching Actions $ACTIONS_VERSION"
	curl -fsSL -o "$work/actions.zip" \
		"https://github.com/Vinillia/actions.ext/releases/download/$ACTIONS_VERSION/actions.ext.zip"
	unzip -oq "$work/actions.zip" -d "$work/prebuilt/actions"
	rm -f "$work/actions.zip"
fi

cp -p "$work/prebuilt/cbasenpc/addons/sourcemod/extensions/cbasenpc.ext.2.tf2.so" \
	"$work/prebuilt/actions/actions.ext/extensions/actions.ext.2.tf2.so" \
	"$out/addons/sourcemod/extensions/"

cp -p "$work/prebuilt/cbasenpc/addons/sourcemod/gamedata/cbasenpc.txt" \
	"$out/addons/sourcemod/gamedata/"
cp "$src/actions/sourcemod/gamedata/actions.games.txt" \
	"$out/addons/sourcemod/gamedata/"

# --- The plugins ---
#
# One include root per project. Never flatten them into a single directory:
# several projects ship a vector.inc, and the wrong one shadows SourceMod's.
compile() {
	name=$(basename "$1" .sp)
	echo "compiling $name"
	"$sm/spcomp64" \
		-i"$sm/include" \
		-i"$src/stocklib" \
		-i"$src/stocksoup-root" \
		-i"$src/cbasenpc/scripting/include" \
		-i"$src/actions/sourcemod/include" \
		-i"$work/ripext/addons/sourcemod/scripting/include" \
		-i"$src/tf2attributes/scripting/include" \
		-i"$src/tf_econ_data/scripting/include" \
		-i"$src/tf2utils/scripting/include" \
		-i"$(dirname "$1")" \
		-o"$out/addons/sourcemod/plugins/$name.smx" \
		"$1" >"$work/$name.log" 2>&1 ||
		{ cat "$work/$name.log"; exit 1; }
}

# The one that is not fetched: this working tree, which is the whole point
compile "$root/source/tf2_defenderbots.sp"
compile "$root/testbed/stats/mvmbots_stats.sp"
compile "$root/testbed/stats/mvmbots_host.sp"
compile "$root/testbed/stats/mvmbots_refund.sp"
compile "$src/tf2attributes/scripting/tf2attributes.sp"
compile "$src/tf_econ_data/scripting/tf_econ_data.sp"
compile "$src/tf2utils/scripting/tf2utils.sp"

cp "$root/gamedata/tf2.defenderbots.txt" \
	"$src/tf2attributes/gamedata/tf2.attributes.txt" \
	"$src/tf_econ_data/gamedata/tf2.econ_data.txt" \
	"$src/tf2utils/gamedata/tf2.utils.nosoop.txt" \
	"$out/addons/sourcemod/gamedata/"

cp -r "$root/configs/defenderbots" "$out/addons/sourcemod/configs/"

# A loadout the run wants instead of the shipped one.
#
# Some faults only exist for a weapon the shipped loadout does not carry. The
# engineer firing a Rescue Ranger at a wall was reported from play and cannot be
# reproduced here at all, because configs/defenderbots/loadout.cfg gives him a
# stock shotgun and the whole code path is behind TF2_IsRescueRangerEquipped.
#
# Overlaying the file rather than editing the shipped one keeps the reproduction
# in the repository next to the thing it reproduces, and keeps a run that forgot
# to set this measuring the loadout everybody actually plays.
if [ -n "${TESTBED_LOADOUT:-}" ]; then
	if [ ! -f "$root/$TESTBED_LOADOUT" ]; then
		echo "TESTBED_LOADOUT: no such file: $root/$TESTBED_LOADOUT" >&2
		exit 1
	fi

	cp "$root/$TESTBED_LOADOUT" "$out/addons/sourcemod/configs/defenderbots/loadout.cfg"
	echo "loadout overlaid from $TESTBED_LOADOUT"
fi

echo "staged into $out"
