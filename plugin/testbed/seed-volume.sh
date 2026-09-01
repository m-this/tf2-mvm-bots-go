#!/bin/sh
# Copy an existing Team Fortress 2 install into the test-bed's volume, so the
# first run does not download fourteen gigabytes of a game this machine already
# has.
#
#   testbed/seed-volume.sh                          from tf2-archipelago
#   testbed/seed-volume.sh some_other_tf2_volume    from somewhere else
#
# The source is opened read only and the server that owns it keeps running.
# A copy rather than a shared mount, deliberately: the test-bed installs its own
# plugins over addons/, and doing that to the volume a live game server is
# reading is how an evening gets ruined.
#
# The game only. Never tf/addons, and this is the whole reason the script is
# careful rather than a one line cp.
#
# Copying it once cost an afternoon. The other server's addons/ carries its own
# SourceMod, its own extensions and the gamedata built against them. Docker only
# seeds a volume from the image when the volume is empty, so a seeded volume
# never got the image's, and the test-bed's entrypoint then installed our
# plugins on top of somebody else's extensions. The pair does not have to match
# and, when it does not, the game server segfaults a minute into every map with
# nothing in any log to say why.
#
# Leave addons/ out and the image seeds it, which is what every other run of
# this server gets.
set -eu

from=${1:-tf2-archipelago_tf2game}
to=${2:-mvmbots-testbed_tf2game}

if ! docker volume inspect "$from" >/dev/null 2>&1; then
	echo "no volume named $from" >&2
	echo "docker volume ls, and pass the one holding tf-dedicated" >&2
	exit 1
fi

if docker volume inspect "$to" >/dev/null 2>&1 &&
	[ -n "$(docker run --rm -v "$to":/to alpine sh -c 'ls /to 2>/dev/null')" ]; then
	echo "$to already has something in it, leaving it alone"
	exit 0
fi

docker volume create "$to" >/dev/null

echo "copying $from into $to, the game only, which is about fourteen gigabytes"

# Ownership matters: the game runs as the image's steam user, and a tree owned
# by root is a server that cannot write its own logs. -a keeps it.
#
# tf/addons and tf/cfg are the source's, not ours. Excluded, so the image seeds
# both from its own layers on first start.
docker run --rm \
	-v "$from":/from:ro \
	-v "$to":/to \
	alpine sh -c '
		cd /from &&
		tar cf - --exclude=./tf/addons --exclude=./tf/cfg . |
			tar xf - -C /to'

echo "seeded $to"
