#!/bin/bash
# Run the mission on a native Linux server rather than in the container.
#
#   testbed/run-native.sh                       one mission on Decoy
#   testbed/run-native.sh --map mvm_coaltown --waves 6
#   testbed/run-native.sh --keep-core           keep the core even if it symbolises
#
# The container test-bed cannot see the thing this is for. A native server is
# reported to crash far more often than the same mod under Docker or on Windows,
# and Docker restarts srcds by itself, so a crash there reads as a hiccup while
# the same crash natively ends the session. This runs the binary directly, with
# cores enabled, and symbolises whatever it leaves behind.
#
# The game tree is a copy, never a share: this installs plugins over addons/ and
# doing that to a tree a live server is reading is how the container test-bed
# once produced five crashes in ten minutes.
set -eu

here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
root=$(CDPATH= cd -- "$here/.." && pwd)

: "${TESTBED_NATIVE_ROOT:=$HOME/tf2-native/tf-dedicated}"
: "${TESTBED_RCONPW:=testbed}"
: "${TESTBED_PORT:=27035}"

map=${TESTBED_MAP:-mvm_decoy}
mission=""
waves=6
timeout=2400
out=""
jump=""
rebuild=1
keep_core=0

while [ $# -gt 0 ]; do
	case "$1" in
	--map) map=$2; shift 2 ;;
	--mission) mission=$2; shift 2 ;;
	--wave) jump=$2; shift 2 ;;
	--waves) waves=$2; shift 2 ;;
	--timeout) timeout=$2; shift 2 ;;
	--out) out=$2; shift 2 ;;
	--no-build) rebuild=0; shift ;;
	--keep-core) keep_core=1; shift ;;
	-h|--help) sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
	*) echo "unknown option: $1" >&2; exit 2 ;;
	esac
done

say() { echo "[native] $*"; }

[ -x "$TESTBED_NATIVE_ROOT/srcds_linux" ] || {
	say "no game at $TESTBED_NATIVE_ROOT"
	say "seed it once: testbed/seed-native.sh"
	exit 1
}

# The same check the container bed makes, for the same reason: a run measured
# under paging measures the machine.
free_mb=$(awk '/MemAvailable/ {print int($2/1024)}' /proc/meminfo 2>/dev/null || echo 99999)

if [ "${TESTBED_MIN_FREE_MB:-1500}" -gt 0 ] && [ "$free_mb" -lt "${TESTBED_MIN_FREE_MB:-1500}" ]; then
	say "only ${free_mb} MB available, and a run under paging measures the machine"
	exit 1
fi

if [ "$rebuild" = 1 ]; then
	say "building"
	"$here/build.sh" >"$here/build/build.log" 2>&1 || { say "build failed, see testbed/build/build.log"; exit 1; }
fi

# One source of truth for the server configuration, shared with the container.
STAGE="$here/build/package"
STEAMAPPDIR="$TESTBED_NATIVE_ROOT"
STEAMAPP=tf
STATS_FILE=${TESTBED_STATS_FILE:-mvmbots_stats.jsonl}
SRCDS_RCONPW=$TESTBED_RCONPW
export STAGE STEAMAPPDIR STEAMAPP STATS_FILE SRCDS_RCONPW

# The same knobs the container reads, so a native run and a container run of the
# same mission differ in the one thing this bed exists to vary.
export BOT_MANAGER_MODE="${TESTBED_BOT_MANAGER_MODE:-1}"
export BOT_TEAM_SIZE="${TESTBED_BOT_TEAM_SIZE:-6}"
export BOT_TEAM_COMP="${TESTBED_BOT_TEAM_COMP:-}"
export BOT_CLASS_BLACKLIST="${TESTBED_BOT_CLASS_BLACKLIST:-}"
export BOT_USE_UPGRADES="${TESTBED_BOT_USE_UPGRADES:-1}"
export BOT_USE_LOADOUTS="${TESTBED_BOT_USE_LOADOUTS:-1}"
export BOT_NEST_RELOCATE="${TESTBED_BOT_NEST_RELOCATE:-0}"
export BOT_FEATURES="${TESTBED_BOT_FEATURES:-}"
export TESTBED_HOST="${TESTBED_HOST:-1}"
export TESTBED_ADMIN_STEAMIDS="${TESTBED_ADMIN_STEAMIDS:-}"
export TESTBED_DEFINE_ONLY=1
# shellcheck disable=SC1090
. "$here/entrypoint.sh"

game="$STEAMAPPDIR/$STEAMAPP"
stats="$game/addons/sourcemod/logs/$STATS_FILE"

install_addons || { say "no sourcemod under $game/addons"; exit 1; }
install_server_cfg
say "installed the bots and the statistics plugin"

# A player's server runs tf2-archipelago's plugin beside the bots, and this bed
# did not, so nothing here could ever see a fault that needs both.
if [ -n "${TESTBED_ARCHIPELAGO_SMX:-}" ]; then
	[ -f "$TESTBED_ARCHIPELAGO_SMX" ] || { say "no plugin at $TESTBED_ARCHIPELAGO_SMX"; exit 1; }
	cp -u "$TESTBED_ARCHIPELAGO_SMX" "$game/addons/sourcemod/plugins/"
	say "and tf2-archipelago, which a player's server also runs"
else
	rm -f "$game/addons/sourcemod/plugins/tf2_archipelago.smx"
fi

rm -f "$stats"

# Cores land in the working directory, which is where the server runs, and the
# soft limit is zero on most machines until something raises it.
ulimit -c unlimited
cores="$TESTBED_NATIVE_ROOT"
rm -f "$cores"/core.* 2>/dev/null || true

say "starting srcds on $map, port $TESTBED_PORT"

# Started with exec so the pid is the server's own, and with stdin closed: srcds
# reads the console, and a backgrounded one sharing the shell's terminal stops on
# SIGTTIN. Its output stops after the Steam banner whatever happens, because it
# takes the console over from there, so readiness is asked of rcon and never of
# this log.
(
	cd "$TESTBED_NATIVE_ROOT"
	exec env LD_LIBRARY_PATH=".:bin:$TESTBED_NATIVE_ROOT/bin:${LD_LIBRARY_PATH:-}" \
		./srcds_linux -game tf -console -usercon -norestart \
		${TESTBED_SRCDS_EXTRA:-} \
		+fps_max 120 -tickrate 66 -ip 0 \
		-port "$TESTBED_PORT" +maxplayers 32 +map "$map" \
		+rcon_password "$TESTBED_RCONPW" +sv_setsteamaccount 0 \
		+servercfgfile server.cfg
) <"/dev/null" >"$here/build/native-server.log" 2>&1 &

server=$!

rcon() { SRCDS_RCONPW=$TESTBED_RCONPW SRCDS_PORT=$TESTBED_PORT python3 "$here/rcon.py" "$@"; }

say "waiting for rcon"
up=0

for _ in $(seq 1 60); do
	if rcon "status" 2>/dev/null | grep -q "^map"; then up=1; break; fi
	kill -0 "$server" 2>/dev/null || break
	sleep 5
done

[ "$up" = 1 ] || {
	say "the server never answered rcon, see testbed/build/native-server.log"
	kill "$server" 2>/dev/null || true
	exit 1
}

if [ "$up" = 1 ] && [ -n "$mission" ]; then
	say "loading mission $mission"
	rcon "tf_mvm_popfile $mission" >/dev/null 2>&1 || true
	sleep 15

	loaded=$(rcon "tf_mvm_popfile" 2>/dev/null | tr -d '\r')

	case "$loaded" in
	*"$mission"*) : ;;
	*) say "the server refused mission $mission and is playing: $loaded"; kill "$server" 2>/dev/null; exit 1 ;;
	esac
fi

if [ "$up" = 1 ]; then
	sleep 20
	rcon "mp_tournament_restart" >/dev/null 2>&1 || true

	if [ -n "$jump" ]; then
		say "jumping to wave $jump"
		sleep 5
		rcon "sv_cheats 1" >/dev/null 2>&1 || true
		rcon "tf_mvm_jump_to_wave $jump" >/dev/null 2>&1 || true
		rcon "sv_cheats 0" >/dev/null 2>&1 || true
	fi
fi

say "watching for $waves wave results, giving up after ${timeout}s"

deadline=$(( $(date +%s) + timeout ))
crashed=0

while [ "$(date +%s)" -lt "$deadline" ]; do
	if ! kill -0 "$server" 2>/dev/null; then
		crashed=1
		break
	fi

	results=$(grep -c '"event":"wave_end"' "$stats" 2>/dev/null | head -1 || true)
	results=${results:-0}

	[ "$results" -ge "$waves" ] && break

	sleep 10
done

kill "$server" 2>/dev/null || true
wait "$server" 2>/dev/null || true

if [ -n "$out" ]; then
	cp "$stats" "$out" 2>/dev/null || true
	say "wrote $out"
fi

core=$(ls -t "$cores"/core.* 2>/dev/null | head -1 || true)

if [ "$crashed" = 1 ] || [ -n "$core" ]; then
	say "the server died"

	if [ -n "$core" ]; then
		say "core at $core, symbolising"
		"$here/symbolise-core.sh" "$core" | tee "$here/build/native-backtrace.txt" | head -30
		[ "$keep_core" = 1 ] || rm -f "$core"
	else
		say "no core: check ulimit -c and /proc/sys/kernel/core_pattern"
		tail -20 "$here/build/native-server.log"
	fi

	exit 1
fi

say "done"
