import glob
import math
import os
import re
"""Say which dispenser spot each authored nest would take, for every map config.

    python3 testbed/checkspots.py

The engineer takes the named dispenser spot that belongs to his nest, where
belonging means the nest nearest to it and within a floor of it. Both of those
are decided by coordinates somebody wrote down after walking a map, and the way
they go wrong is silent: the spot is ignored and the dispenser goes in the nest
area instead, which looks fine and quietly throws away the walking.

So this reads the configs and says, per nest, which spot it ends up with and
which it refused and why. A nest that falls back on every map in the file is a
nest whose spots were authored for a different pairing, and Mannhattan is the
example: all three of its spots sit the better part of a storey below the nests
that are nearest to them.

It reads the config files and nothing else, so it costs nothing and needs no
server.
"""



def tokens(text):
    text = re.sub(r'//[^\n]*', '', text)
    return re.findall(r'"([^"]*)"|([{}])', text)

def parse(text):
    toks = [a if b == "" else b for a, b in tokens(text)]
    pos = 0
    def block():
        nonlocal pos
        out = []
        while pos < len(toks):
            t = toks[pos]
            if t == '}':
                pos += 1
                return out
            key = t; pos += 1
            if pos < len(toks) and toks[pos] == '{':
                pos += 1
                out.append((key, block()))
            else:
                out.append((key, toks[pos])); pos += 1
        return out
    while pos < len(toks) and toks[pos] != '{':
        pos += 1
    pos += 1
    return block()

def find(node, name):
    for k, v in node:
        if k == name and isinstance(v, list):
            return v
    return None

def points(node, name):
    """Every origin under a block, with the zone beside it when the map names one."""
    b = find(node, name)
    if b is None:
        return []
    out = []
    for _, entry in b:
        if not isinstance(entry, list):
            continue
        fields = dict((k, v) for k, v in entry if not isinstance(v, list))
        if 'origin' in fields:
            out.append((tuple(float(x) for x in fields['origin'].split()),
                        fields.get('zone', '')))
    return out

MATCH = 400.0
d = math.dist

for path in sorted(glob.glob("configs/defenderbots/map/*.cfg")):
    root = parse(open(path).read())
    allnests = points(root, 'EngineerNest') + points(root, 'NestTankOnly') + points(root, 'NestNoTank')
    spots = points(root, 'DispenserSpot')
    if not allnests or not spots:
        continue
    print(f"{os.path.basename(path)}: {len(allnests)} nests, {len(spots)} dispenser spots")
    for i, (nest, nest_zone) in enumerate(allnests, 1):
        # His own zone if the map put a spot in it, the unreserved ones if it did not
        owned = [(round(d(s, nest)), s) for s, z in spots if z == nest_zone]

        if not owned and nest_zone:
            owned = [(round(d(s, nest)), s) for s, z in spots if not z]

        note = ""
        if owned:
            best = min(owned)
            print(f"  nest {i} {nest} zone={nest_zone!r} -> {best[1]} at {best[0]}u{note}")
        else:
            print(f"  nest {i} {nest} zone={nest_zone!r} -> falls back to the nest area{note}")
