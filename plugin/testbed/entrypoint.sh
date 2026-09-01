#!/bin/bash
# Install the staged tree into the game volume, write the server configuration,
# then hand over to the image's own entrypoint.
#
# SourceMod is downloaded by that entrypoint, into the volume, on first start,
# so the install waits for it in the background rather than racing it. It keeps
# watching for the life of the container because the game auto-updates and
# SourceMod can be reinstalled under it.
set -eu

# Defaulted rather than fixed, because run-native.sh sources this to write the
# same server.cfg into a game tree on the host rather than in the image.
STAGE="${STAGE:-/opt/mvmbots}"
GAME="${GAME:-${STEAMAPPDIR}/${STEAMAPP}}"
INTERVAL=30

install_addons() {
	[ -d "$GAME/addons/sourcemod" ] || return 1

	# -u, and the whole test-bed depends on it. This runs every thirty seconds
	# for as long as the server lives, and cp truncates the destination before
	# it writes. Truncating a file the running server has mapped invalidates
	# the pages under it: the next instruction the game executes out of that
	# extension is SIGBUS, or SIGSEGV once a half written one is executing.
	#
	# It reads as a crash in whichever extension happened to be rewritten, on a
	# stripped binary, every thirty to sixty seconds, with no plugin loaded and
	# nobody connected. It cost a day. -u compares the timestamp, the staged
	# tree never changes, so nothing is rewritten after the first copy.
	cp -ru "$STAGE/addons/." "$GAME/addons/"
	mkdir -p "$GAME/addons/sourcemod/logs"

	# A seeded volume comes from a server that was running something. The
	# Archipelago plugin locks classes and weapon slots until a multiworld
	# hands them over, which is a fine game and a ruined measurement: the bots
	# would be measured playing with half a loadout for reasons this test-bed
	# knows nothing about.
	rm -f "$GAME/addons/sourcemod/plugins/tf2_archipelago.smx" \
		"$GAME/cfg/sourcemod/tf2_archipelago.cfg" 2>/dev/null || true

	# SourceMod loads every plugin in plugins/. The stack that comes with the
	# image is not what is being measured, and a plugin that touches the game
	# is a variable in the experiment.
	rm -f "$GAME/addons/sourcemod/plugins/funcommands.smx" \
		"$GAME/addons/sourcemod/plugins/playercommands.smx" \
		"$GAME/addons/sourcemod/plugins/basecomm.smx" \
		"$GAME/addons/sourcemod/plugins/nextmap.smx" \
		"$GAME/addons/sourcemod/plugins/basvotes.smx" \
		"$GAME/addons/sourcemod/plugins/mapchooser.smx" \
		"$GAME/addons/sourcemod/plugins/rockthevote.smx" 2>/dev/null || true

	return 0
}

# Whoever walks the map to write down nest spots joins as a plain player, and
# sm_dump_spot needs ADMFLAG_GENERIC. Two things grant it, because they fail in
# different ways.
#
# admins_simple.ini is the real one: named SteamIDs get root, so sm_addbots and
# the rest work too. It depends on the client's SteamID reaching the server,
# which this server has no Steam session to verify.
#
# The override is the fallback, and only for sm_dump_spot: it prints a position
# and changes nothing, so a server on a LAN losing that flag costs nothing.
install_admin_config() {
	dir="$GAME/addons/sourcemod/configs"
	[ -d "$dir" ] || return 1

	{
		echo '// Managed by the mvm-bots test-bed. Edits here are replaced on restart.'
		for id in $(echo "${TESTBED_ADMIN_STEAMIDS:-}" | tr ',' ' '); do
			# admins_simple.ini reads STEAM_0:Y:Z and [U:1:N] and nothing else. A
			# SteamID64 in it is not an error, it is a line that never matches
			# anybody, so the account number comes out of it here and both forms
			# get written. Which one the client reports is the server's business.
			case "$id" in
			7656119*)
				account=$((id - 76561197960265728))
				printf '"STEAM_0:%d:%d" "99:z"\n' "$((account % 2))" "$((account / 2))"
				printf '"[U:1:%d]" "99:z"\n' "$account"
				;;
			*)
				printf '"%s" "99:z"\n' "$id"
				;;
			esac
		done
	} >"$dir/admins_simple.ini"

	cat >"$dir/admin_overrides.cfg" <<-CFG
	// Managed by the mvm-bots test-bed. Edits here are replaced on restart.
	Overrides
	{
		"sm_dump_spot"		""
	}
	CFG

	return 0
}

# The measurements are worth nothing if the run they came from cannot be
# described, so the configuration is generated here in one place and the run
# script prints it back.
install_server_cfg() {
	target="$GAME/cfg/server.cfg"
	[ -d "$(dirname "$target")" ] || return 1

	# One convar line per name=value in BOT_FEATURES, so a run can turn a
	# behaviour off without a rebuild and the results say which was on.
	BOT_FEATURE_LINES=""

	for pair in $(echo "${BOT_FEATURES:-}" | tr ',' ' '); do
		name=${pair%%=*}
		value=${pair#*=}

		[ -n "$name" ] || continue

		BOT_FEATURE_LINES="${BOT_FEATURE_LINES}sm_redbots_feature_${name} ${value}
	"
	done

	# The same again for convars that are not feature switches. Some things a
	# run wants to vary are a number rather than an on or an off, and the three
	# BLU scales are the first of them: 1.0 is already the off, so a second
	# switch beside them would be two ways to say one thing.
	for pair in $(echo "${BOT_CVARS:-}" | tr ',' ' '); do
		name=${pair%%=*}
		value=${pair#*=}

		[ -n "$name" ] || continue

		BOT_FEATURE_LINES="${BOT_FEATURE_LINES}${name} ${value}
	"
	done

	cat >"$target" <<-CFG
	// Managed by the mvm-bots test-bed. Edits here are replaced on restart.
	hostname "MvM defender bots test-bed"
	rcon_password "${SRCDS_RCONPW:-testbed}"
	sv_password ""
	sv_lan 1

	// Marking up a map means flying it: noclip is how the EngineerNest and
	// TeleporterExit spots in configs/defenderbots/map get written down.
	sv_cheats 1

	// The log file, so chat from whoever is walking the map can be read from
	// outside the game. sv_logecho 0 keeps it out of the container's stdout,
	// which the supervisor below already writes to every thirty seconds.
	sv_logfile 1
	sv_logecho 0
	log on

	// Nobody is going to join, so nothing waits for anybody. One ready player
	// starts a wave, and the bots ready themselves up.
	tf_mvm_min_players_to_start 1

	// An empty server hibernates: it stops simulating, no timer runs, and
	// nothing ever adds a bot. Every server this mod runs on has a person on
	// it keeping it awake. This one never will, so hibernation has to go, and
	// without it there is no test-bed at all.
	//
	// tf_allow_server_hibernation, not sv_hibernate_when_empty: the generic
	// Source convar does not exist in Team Fortress 2, and setting it is a
	// line in a config file that quietly does nothing.
	tf_allow_server_hibernation 0
	mp_idlemaxtime 0
	mp_idledealmethod 0
	sv_pure 0
	sv_pausable 0
	setpause 0

	// The bots. Mode 1 is READY_BOTS: RED is filled between waves, which is
	// what lets a wave start with no human on the server at all. Mode 2 is
	// AUTO_BOTS, which fills when the wave begins, and is what tf2-archipelago
	// runs: BOT_MANAGER_MODE=2 plays a mission the way a player's server does.
	//
	// min_players -1 turns off the mod's own ready-up gate, which otherwise
	// counts RED before the bots exist and blocks the ready that would have
	// spawned them.
	sm_redbots_manager_mode ${BOT_MANAGER_MODE:-1}
	sm_redbots_manager_defender_team_size ${BOT_TEAM_SIZE:-6}
	sm_redbots_manager_min_players -1
	sm_redbots_manager_kick_bots 0
	sm_redbots_manager_bot_use_upgrades ${BOT_USE_UPGRADES:-1}
	sm_redbots_manager_engineer_nest_relocate ${BOT_NEST_RELOCATE:-0}
	sm_redbots_manager_use_custom_loadouts ${BOT_USE_LOADOUTS:-1}

	// Feature switches, for running the same mission twice with one thing
	// different. BOT_FEATURES is a list like "spy_glance=0,sticky_stack=0"
	${BOT_FEATURE_LINES}
	sm_redbots_manager_class_blacklist "${BOT_CLASS_BLACKLIST:-}"
	sm_redbots_manager_team_composition "${BOT_TEAM_COMP:-}"

	// The fake client that holds a seat on RED and readies up. Without it
	// nothing ever starts a wave: the mod adds its bots in response to a human
	// pressing F4, and the game will not begin a wave with nobody ready.
	mvmbots_host_enabled ${TESTBED_HOST:-1}

	// Where the wave results are written. The run script reads this file.
	mvmbots_stats_path "logs/${STATS_FILE:-mvmbots_stats.jsonl}"
	CFG

	return 0
}

supervise() {
	while true; do
		if install_addons && install_server_cfg && install_admin_config; then
			echo "[test-bed] installed the bots and the statistics plugin"
		fi
		sleep "$INTERVAL"
	done
}

# run-native.sh sources this for the installers above and runs the server itself,
# so that the native test-bed and the container write the same server.cfg. Two
# copies of that file would drift, and a difference between the two beds is the
# one thing this is for.
if [ -n "${TESTBED_DEFINE_ONLY:-}" ]; then
	return 0
fi

supervise &

# The image's own entrypoint owns the command line, and reads SRCDS_STARTMAP,
# SRCDS_MAXPLAYERS and the rest from the environment. The mission is not set
# here: tf_mvm_popfile only works once a map is loaded, so run.sh sends it.
exec bash "${HOMEDIR}/entry.sh"
