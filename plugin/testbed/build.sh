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

# $1 repo, $2 ref (a tag, a branch or a full commit), $3 directory
#
# Stamped with the ref it holds, the way the compiler is: $work survives
# between runs, and a checkout left from the previous pin builds the old
# version under the new name.
fetch() {
	stamp="$src/$3.ref"
	if [ -d "$src/$3" ]; then
		[ "$(cat "$stamp" 2>/dev/null)" = "$2" ] && return 0
		rm -rf "$src/$3"
	fi
	echo "fetching $1@$2"
	mkdir -p "$src/$3"
	git -C "$src/$3" init --quiet
	git -C "$src/$3" fetch --quiet --depth 1 "https://github.com/$1" "$2"
	git -C "$src/$3" checkout --quiet FETCH_HEAD
	printf '%s\n' "$2" >"$stamp"
}

# patches/<directory>/*.patch, applied in order. Already applied is fine, a
# second run finds its own work; not applying is a failure, because it means
# upstream moved under the pin.
apply_patches() {
	for patch in "$root/testbed/patches/$1"/*.patch; do
		[ -e "$patch" ] || return 0
		git -C "$src/$1" apply --reverse --check "$patch" 2>/dev/null && continue
		echo "applying $1/$(basename "$patch")"
		git -C "$src/$1" apply --whitespace=nowarn "$patch"
	done
}

# --- Sources the mod compiles against ---

fetch "$TF2ATTRIBUTES_REPO" "$TF2ATTRIBUTES_VERSION" tf2attributes
fetch nosoop/SM-TFEconData "$TFECONDATA_VERSION" tf_econ_data
fetch nosoop/SM-TFUtils "$TF2UTILS_VERSION" tf2utils
fetch nosoop/stocksoup "$STOCKSOUP_REF" stocksoup-root/stocksoup
fetch TF2-DMB/CBaseNPC "$CBASENPC_VERSION" cbasenpc
fetch Vinillia/actions.ext "$ACTIONS_COMMIT" actions

# What each patch is for is in patches/README.md.
apply_patches tf2attributes
apply_patches tf2utils
apply_patches cbasenpc
apply_patches actions

# --- The compiler ---
#
# Stamped with the version it holds, because $work survives between runs: a
# compiler left from the previous pin compiles and says nothing, so a bump would
# take effect the first time somebody cleaned the directory and not before.
spcomp_stamp="$work/spcomp.ref"
if [ ! -d "$work/spcomp" ] || [ "$(cat "$spcomp_stamp" 2>/dev/null)" != "$SOURCEMOD_VERSION" ]; then
	echo "fetching SourceMod $SOURCEMOD_VERSION"
	rm -rf "$work/spcomp"
	mkdir -p "$work/spcomp"
	curl -fsSL -o "$work/sourcemod.tar.gz" \
		"https://sm.alliedmods.net/smdrop/$SOURCEMOD_BRANCH/sourcemod-$SOURCEMOD_VERSION-linux.tar.gz"
	echo "$SOURCEMOD_SHA256  $work/sourcemod.tar.gz" | sha256sum -c -
	tar xzf "$work/sourcemod.tar.gz" -C "$work/spcomp"
	rm -f "$work/sourcemod.tar.gz"
	printf '%s\n' "$SOURCEMOD_VERSION" >"$spcomp_stamp"
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
#
# Actions v4.0.1 is built for SourceMod 1.13 (extension API 9). CBaseNPC has no
# such release: 1.15.4.126 is API 8, and a 1.13 server refuses to load it. A
# bed that runs this mod on 1.13 needs the CBaseNPC tf2-archipelago builds from
# source (deploy/bots/build-extensions.sh), and its DHooks too: the drop's never
# hands a virtual hook's Address to the plugin correctly (its patch
# sourcemod/0001). Compiling, which is what make check asks of this script, needs
# neither.

cbasenpc_prebuilt="$work/prebuilt/cbasenpc-$CBASENPC_VERSION"
actions_prebuilt="$work/prebuilt/actions-$ACTIONS_VERSION"

if [ ! -d "$cbasenpc_prebuilt" ]; then
	echo "fetching CBaseNPC $CBASENPC_VERSION"
	mkdir -p "$cbasenpc_prebuilt"
	curl -fsSL "https://github.com/TF2-DMB/CBaseNPC/releases/download/$CBASENPC_VERSION/cbasenpc${CBASENPC_VERSION}_linux.tar.gz" |
		tar xz -C "$cbasenpc_prebuilt"
fi

if [ ! -d "$actions_prebuilt" ]; then
	echo "fetching Actions $ACTIONS_VERSION"
	curl -fsSL -o "$work/actions.zip" \
		"https://github.com/Vinillia/actions.ext/releases/download/$ACTIONS_VERSION/actions.ext.zip"
	echo "$ACTIONS_SHA256  $work/actions.zip" | sha256sum -c -
	mkdir -p "$actions_prebuilt"
	unzip -oq "$work/actions.zip" -d "$actions_prebuilt"
	rm -f "$work/actions.zip"
fi

cp -p "$cbasenpc_prebuilt/addons/sourcemod/extensions/cbasenpc.ext.2.tf2.so" \
	"$actions_prebuilt/extensions/actions.ext.2.tf2.so" \
	"$out/addons/sourcemod/extensions/"

cp -p "$cbasenpc_prebuilt/addons/sourcemod/gamedata/cbasenpc.txt" \
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
