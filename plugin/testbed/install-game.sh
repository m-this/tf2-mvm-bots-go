#!/bin/sh
# Put a Team Fortress 2 dedicated server in the test-bed's volume.
#
#   testbed/install-game.sh                 into mvmbots-testbed_tf2game
#   testbed/install-game.sh some_volume     into somewhere else
#
# The game image carries SourceMod and the server scripts, not the game: it
# expects the fifteen gigabytes to be in the volume already. Nothing downloaded
# them, because every run so far seeded the volume from another server. So the
# first clean run found no srcds_run and exited 127.
#
# This is the download, from Steam, with a validate. It is the same install the
# ansible role's stack ends up with, and it is the only thing that should ever
# write addons/ besides the image itself.
set -eu

to=${1:-mvmbots-testbed_tf2game}

docker volume create "$to" >/dev/null

if docker run --rm -v "$to":/v alpine test -x /v/srcds_run; then
	echo "$to already has the game"
	exit 0
fi

echo "downloading Team Fortress 2 into $to, about fifteen gigabytes"

# Two steamcmd runs, and the first is not redundant. On a first ever run
# steamcmd has no application info cached, and app_update answers
# "Missing configuration" and installs nothing at all rather than failing
# loudly. Fetching the app info first is what makes the second run work.
#
# As root to fix the volume's ownership, then as steam, because steamcmd
# refuses to run as root and the game runs as the image's steam user.
docker run --rm --user 0 -v "$to":/data cm2network/steamcmd:root bash -c '
	chown -R steam:steam /data
	su steam -c "
		/home/steam/steamcmd/steamcmd.sh +login anonymous \
			+app_info_update 1 +app_info_print 232250 +quit >/dev/null
		/home/steam/steamcmd/steamcmd.sh +force_install_dir /data \
			+login anonymous +app_update 232250 validate +quit
	"'

docker run --rm -v "$to":/v alpine test -x /v/srcds_run

echo "installed the game in $to"
