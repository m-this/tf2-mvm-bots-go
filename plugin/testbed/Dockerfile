# A Team Fortress 2 server running this working tree's bots and nothing else.
#
# The game files and SourceMod are downloaded at runtime into the volume at
# /home/steam/tf-dedicated. A file baked into a path a volume later covers is a
# file nobody sees, so the staged tree goes to /opt and the entrypoint installs
# it once SourceMod exists.

FROM debian:stable-slim AS bots

RUN apt-get update \
	&& apt-get install --yes --no-install-recommends \
		git curl ca-certificates unzip \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /repo

# The mod, its configuration and its gamedata are copied rather than fetched:
# the test-bed measures what is in the tree, not what is in a tag.
COPY testbed/versions.env testbed/versions.env
COPY testbed/build.sh testbed/build.sh
COPY testbed/stats testbed/stats
COPY source source
COPY configs configs
COPY gamedata gamedata

RUN sh testbed/build.sh


FROM cm2network/tf2:sourcemod

USER root

RUN mkdir -p /opt/mvmbots
COPY --from=bots /repo/testbed/build/package/addons /opt/mvmbots/addons
COPY testbed/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chown -R steam:steam /opt/mvmbots \
	&& chmod +x /usr/local/bin/entrypoint.sh

USER steam
WORKDIR /home/steam
CMD ["bash", "/usr/local/bin/entrypoint.sh"]
