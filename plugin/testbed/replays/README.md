# Replays

A player's `server.cfg`, out of a debug bundle, for `testbed -replay`.

This directory is ignored. That is the point of it: a `server.cfg` carries the
player's `rcon_password` and `sv_password`, and one of them reached a public
repository because the file was put beside the compose file and a later
`git add -A testbed/` took it. The runner refuses a `-replay` path that git
would commit, so this is where one goes.

Only the settings in `settingsReplayed` are read; the rest of the file, the
credentials included, never leaves it.
