/*
Package lineoffire is the part of source/redbots3/util.sp that asks whether a bot
can shoot at something without hitting the world.

The trace is the game's own, with the filter the game's bots use: it ignores the
people in the way and the team's own buildings, because neither is a reason to
hold fire.
*/
package lineoffire

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
TraceFilterTFBot is the filter, which is two of the game's own chained.

NextBotTraceFilterIgnoreActors drops anybody standing in the way, and
CTraceFilterIgnoreFriendlyCombatItems drops the team's own buildings and
projectiles. What is left is the world, which is the only thing worth not
shooting through.
*/
//
//sp:name TraceFilter_TFBot
func TraceFilterTFBot(entity int32, contentsMask int32, data engine.Properties) bool {
	// NextBotTraceFilterIgnoreActors
	if engine.EntityOf(entity).IsCombatCharacter() {
		return false
	}

	/* CTraceFilterIgnoreFriendlyCombatItems

	The shipped file initialises the pass entity to -1 before the lookup, so a
	map without the key skips the entity filter rather than filtering against
	entity zero. GetValue only writes when the key is there, so the initial
	value is the answer for a missing one. */
	foundPass, passEnt := data.Value("m_pPassEnt")

	if !foundPass {
		passEnt = -1
	}

	_, collisionGroup := data.Value("m_collisionGroup")

	_, ignoreTeam := data.Value("m_iIgnoreTeam")

	if engine.IsCombatItem(entity) {
		if engine.EntityTeamNumber(entity) == ignoreTeam {
			return false
		}

		// m_bCallerIsProjectile is false here
	}

	// CTraceFilterSimple as BaseClass of CTraceFilterIgnoreFriendlyCombatItems
	if !engine.StandardFilterRules(entity, contentsMask) {
		return false
	}

	if passEnt != -1 {
		if !engine.PassServerEntityFilter(entity, passEnt) {
			return false
		}
	}

	if !engine.ShouldCollide(entity, collisionGroup, contentsMask) {
		return false
	}

	if !engine.GameRulesShouldCollide(collisionGroup, engine.CollisionGroupOf(entity)) {
		return false
	}

	// CTraceFilterChain checks if both filters are true
	return true
}

// IsLineOfFireClearPosition says nothing solid stands between the two points.
//
//sp:name IsLineOfFireClearPosition
//sp:const from
//sp:const to
func IsLineOfFireClearPosition(client int32, from [3]float32, to [3]float32) bool {
	properties := engine.NewProperties()

	properties.SetProperty("m_pPassEnt", client)
	properties.SetProperty("m_collisionGroup", engine.CollisionGroupNone())
	properties.SetProperty("m_iIgnoreTeam", engine.GetClientTeam(client))

	engine.TraceRayFilterData(from, to, engine.MaskSolidBrushOnly(), engine.RayTypeEndPoint(), TraceFilterTFBot, properties)

	properties.Close()

	return !engine.DidHit()
}

// IsLineOfFireClearEntity is the same question about a target, where hitting the
// target itself is a clear line.
//
//sp:name IsLineOfFireClearEntity
//sp:const from
func IsLineOfFireClearEntity(client int32, from [3]float32, who int32) bool {
	properties := engine.NewProperties()

	properties.SetProperty("m_pPassEnt", client)
	properties.SetProperty("m_collisionGroup", engine.CollisionGroupNone())
	properties.SetProperty("m_iIgnoreTeam", engine.GetClientTeam(client))

	engine.TraceRayFilterData(from, engine.WorldSpaceCenter(who), engine.MaskSolidBrushOnly(), engine.RayTypeEndPoint(), TraceFilterTFBot, properties)

	properties.Close()

	return !engine.DidHit() || engine.TraceEntityIndex() == who
}

// FindOnlyOneVisibleEntity is whichever of the two can be seen when the other
// cannot, and -2 when both can.
//
//sp:name FindOnlyOneVisibleEntity
func FindOnlyOneVisibleEntity(client int32, ent1 int32, ent2 int32) int32 {
	if !IsLineOfFireClearEntity(client, engine.EyePosition(client), ent1) {
		return ent2
	}

	if !IsLineOfFireClearEntity(client, engine.EyePosition(client), ent2) {
		return ent1
	}

	return -2
}
