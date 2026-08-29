package body_test

import (
	"fmt"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
	"github.com/m-this/tf2-mvm-bots-go/internal/sp"
)

// The buildings the two object scans walk. Entity indexes start above the
// client slots, the way a server's do, so a scan that confused an entity index
// with a client slot reads a building that is not there.
const firstEntity = 100

// entity is one building, and everything the two scans ask about it.
type entity struct {
	class   string
	object  engine.Object
	team    int32
	placing bool
	carried bool
	sapped  bool
	origin  [3]float32
	// The two the currency pack scan asks about, which no building has and
	// every pack does.
	distributed bool
	onGround    bool
}

// cannedEntities is one of each thing the scans skip, plus several they do not,
// so every continue in both loops is taken by something.
func cannedEntities() []entity {
	return []entity{
		{class: "obj_sentrygun", object: 2, team: 3, origin: [3]float32{100, 0, 0}},
		{class: "obj_sapper", object: engineObjectSapper, team: 3, origin: [3]float32{110, 0, 0}},
		{class: "obj_teleporter", object: 1, team: 2, origin: [3]float32{120, 0, 0}},
		{class: "obj_teleporter", object: 1, team: 3, placing: true, origin: [3]float32{130, 0, 0}},
		{class: "obj_teleporter", object: 1, team: 3, carried: true, origin: [3]float32{140, 0, 0}},
		{class: "obj_teleporter", object: 1, team: 3, sapped: true, origin: [3]float32{150, 0, 0}},
		{class: "obj_teleporter", object: 1, team: 3, origin: [3]float32{160, 0, 0}},
		{class: "obj_dispenser", object: 0, team: 3, origin: [3]float32{170, 0, 0}},
		// Two buildings in the same place, so <= and < on bestDistance
		// are told apart here as well as in the client scans.
		{class: "obj_dispenser", object: 0, team: 3, origin: [3]float32{170, 0, 0}},
		// The money, which the currency pack scan walks and the building
		// scans skip: one already handed out, one still in the air, and
		// two that can be picked up.
		{class: "item_currency_pack_small", origin: [3]float32{180, 0, 0}, onGround: true},
		{class: "item_currency_pack_small", origin: [3]float32{190, 0, 0}, distributed: true, onGround: true},
		{class: "item_currency_pack_large", origin: [3]float32{200, 0, 0}},
		{class: "item_currency_pack_large", origin: [3]float32{210, 0, 0}, onGround: true},
	}
}

// engineObjectSapper is TFObject_Sapper, held here rather than called, because
// the extern's Go body is the constant and a test that called it would be
// asserting the constant against itself.
const engineObjectSapper = engine.Object(3)

// findEntityByClassname is the engine's walk: the next entity after start whose
// class matches, and -1 when there is none.
func findEntityByClassname(entities []entity, start int32, pattern string) int32 {
	next := int(start-firstEntity) + 1
	if start < firstEntity {
		next = 0
	}
	for i := next; i < len(entities); i++ {
		if classMatches(pattern, entities[i].class) {
			return firstEntity + int32(i)
		}
	}
	return -1
}

// classMatches is SourceMod's wildcard, which is a trailing star and nothing
// else. Written out in both languages, because the standalone SourcePawn has no
// string functions at all.
func classMatches(pattern, name string) bool {
	if star := strings.IndexByte(pattern, '*'); star >= 0 {
		return strings.HasPrefix(name, pattern[:star])
	}
	return pattern == name
}

func entityAt(entities []entity, index int32) entity {
	return entities[index-firstEntity]
}

// entityInclude writes the buildings and the stubs that answer about them.
func entityInclude(entities []entity) string {
	var b strings.Builder
	b.WriteString(`
enum TFObjectType
{
	TFObject_Dispenser = 0,
	TFObject_Teleporter,
	TFObject_Sentry,
	TFObject_Sapper
};

`)
	fmt.Fprintf(&b, "#define FIRST_ENTITY %d\n#define ENTITY_COUNT %d\n\n", firstEntity, len(entities))

	b.WriteString("char gEntClass[ENTITY_COUNT][32] =\n{\n")
	for _, e := range entities {
		fmt.Fprintf(&b, "\t%q,\n", e.class)
	}
	b.WriteString("};\n")

	writeInts(&b, "gEntObject", intsOf(entities, func(e entity) int32 { return int32(e.object) }))
	writeInts(&b, "gEntTeam", intsOf(entities, func(e entity) int32 { return e.team }))
	writeEntBools(&b, "gEntPlacing", entities, func(e entity) bool { return e.placing })
	writeEntBools(&b, "gEntCarried", entities, func(e entity) bool { return e.carried })
	writeEntBools(&b, "gEntSapped", entities, func(e entity) bool { return e.sapped })
	writeEntBools(&b, "gEntDistributed", entities, func(e entity) bool { return e.distributed })
	writeEntBools(&b, "gEntOnGround", entities, func(e entity) bool { return e.onGround })

	b.WriteString("float gEntOrigin[ENTITY_COUNT][3] =\n{\n")
	for _, e := range entities {
		fmt.Fprintf(&b, "\t{%s, %s, %s},\n", sp.FloatLiteral(e.origin[0]), sp.FloatLiteral(e.origin[1]), sp.FloatLiteral(e.origin[2]))
	}
	b.WriteString("};\n")

	b.WriteString(`
/* SourceMod's wildcard is a trailing star and nothing else, written out because
   the standalone SourcePawn has no string functions at all. */
stock bool ClassMatches(const char[] pattern, const char[] name)
{
	for (int i = 0; i < 32; i++)
	{
		if (pattern[i] == '*')
			return true;

		if (pattern[i] != name[i])
			return false;

		if (pattern[i] == 0)
			return true;
	}
	return true;
}

stock int FindEntityByClassname(int start, const char[] classname)
{
	int next = start < FIRST_ENTITY ? 0 : start - FIRST_ENTITY + 1;

	for (int i = next; i < ENTITY_COUNT; i++)
	{
		if (ClassMatches(classname, gEntClass[i]))
			return FIRST_ENTITY + i;
	}
	return -1;
}

stock void BaseEntity_GetAbsOrigin(int entity, float origin[3])
{
	for (int axis = 0; axis < 3; axis++)
		origin[axis] = gEntOrigin[entity - FIRST_ENTITY][axis];
}

`)
	fmt.Fprintf(&b, "stock TFObjectType TF2_GetObjectType(int entity) { Trace(%d, entity); return view_as<TFObjectType>(gEntObject[entity - FIRST_ENTITY]); }\n", traceObjectType)
	fmt.Fprintf(&b, "stock int BaseEntity_GetTeamNumber(int entity) { Trace(%d, entity); return gEntTeam[entity - FIRST_ENTITY]; }\n", traceEntityTeamNumber)
	fmt.Fprintf(&b, "stock bool TF2_IsPlacing(int entity) { Trace(%d, entity); return gEntPlacing[entity - FIRST_ENTITY]; }\n", traceIsPlacing)
	fmt.Fprintf(&b, "stock bool TF2_IsCarried(int entity) { Trace(%d, entity); return gEntCarried[entity - FIRST_ENTITY]; }\n", traceIsCarried)
	fmt.Fprintf(&b, "stock bool TF2_HasSapper(int entity) { Trace(%d, entity); return gEntSapped[entity - FIRST_ENTITY]; }\n", traceHasSapper)
	b.WriteString("\n#define FL_ONGROUND 1\n\n")
	fmt.Fprintf(&b, "stock int GetEntProp(int entity, PropType propType, const char[] prop) { Trace(%d, entity); return gEntDistributed[entity - FIRST_ENTITY] ? 1 : 0; }\n", traceEntProp)
	fmt.Fprintf(&b, "stock int GetEntityFlags(int entity) { Trace(%d, entity); return gEntOnGround[entity - FIRST_ENTITY] ? FL_ONGROUND : 0; }\n", traceEntityFlags)
	fmt.Fprintf(&b, "stock bool BaseEntity_IsPlayer(int entity) { Trace(%d, entity); return entity <= WORLD_SLOTS; }\n", traceIsPlayer)
	fmt.Fprintf(&b, "stock int TF2_GetNumHealers(int client) { Trace(%d, client); return client %% 3; }\n", traceNumHealers)
	fmt.Fprintf(&b, "stock int TF2Util_GetPlayerHealer(int client, int index) { Trace(%d, client); return (client + index) %% WORLD_SLOTS + 1; }\n", tracePlayerHealer)
	return b.String()
}

func intsOf(entities []entity, of func(entity) int32) []int32 {
	out := make([]int32, len(entities))
	for i, e := range entities {
		out[i] = of(e)
	}
	return out
}

func writeEntBools(b *strings.Builder, name string, entities []entity, of func(entity) bool) {
	values := make([]bool, len(entities))
	for i, e := range entities {
		values[i] = of(e)
	}
	writeBools(b, name, values)
}
