/*
Package engine is what a generated body calls when it calls the engine.

One Go function per engine call, each carrying the directive that says how the
call is written in SourcePawn. internal/spbody reads those directives, so the
Go a body compiles against and the SourcePawn it emits come from the same
declaration and cannot drift apart.

Three kinds of call, and the directive names which:

	//sp:native NAME     a SourceMod native, called by name
	//sp:method NAME     a methodmap method, written on its receiver
	//sp:new NAME        a constructor, new NAME()
	//sp:property NAME   a methodmap property, written without parentheses
	//sp:delete NAME     the delete of a handle, which makes its type a lifetime
	//sp:slot NAME       a plugin array indexed by its first argument
	//sp:slotset NAME    the same array, written to
	//sp:body NAME       SourcePawn this port already generates
	//sp:global NAME     a SourcePawn variable, not a call at all
	//sp:plugin NAME     a plugin function the port has not reached yet
	//sp:sdkcall HANDLE  SDKCall through a handle prepared at load
	//sp:address NAME    a read through a raw address

A directive may end in the flag "returns", which says the SourcePawn declaration
returns its array rather than filling a parameter with it. That is the float[]
form, and spcomp will not assign one to a sized array, so neither will this: such
a call can only be an argument to something else.

A plugin extern is temporary by construction. It names SourcePawn this
repository is going to own, and when it does the extern goes and the call
becomes an ordinary one. internal/body refuses an extern that names a function a
body already generates, so the two can never both be there.

In a Go process none of them mean anything, so the default answer to all of them
is a panic. Install puts a set of answers behind them, which is what the
differential test does: the same canned answers on both sides, and the two call
traces have to match.
*/
package engine

import "fmt"

// Class is SourceMod's TFClassType, named here so a ported signature keeps the
// tag its callers pass rather than widening to int.
//
//sp:tag TFClassType
type Class int32

// Team is SourceMod's TFTeam, for the same reason.
//
//sp:tag TFTeam
type Team int32

// Condition is SourceMod's TFCond.
//
//sp:tag TFCond
type Condition int32

// Object is SourceMod's TFObjectType.
//
//sp:tag TFObjectType
type Object int32

// PropType is SourceMod's PropType, the networked or the datamap table.
//
//sp:tag PropType
type PropType int32

// Weapon is SourceMod's TFWeaponType.
//
//sp:tag TFWeaponType
type Weapon int32

// Calls is the set of answers a body gets. A nil field is a call the caller did
// not expect the body to make, and reaching it is a failed expectation rather
// than a zero value quietly standing in for one.
type Calls struct {
	IsClientInGame func(client int32) bool
	IsPlayerAlive  func(client int32) bool
	GetClientTeam  func(client int32) int32
	HasAmmo        func(weapon int32) bool
	Clip1          func(weapon int32) int32
	Origin         func(client int32) [3]float32
	MaxClients     func() int32
	VectorDistance func(a [3]float32, b [3]float32) float32

	// The plugin's own, still hand-written SourcePawn. Each one is work the
	// port has not reached, and each goes the day it does. The TF2_ names
	// beside them are not: they come from SourceMod and from the vendored
	// stocklib include, which this repository is not going to rewrite.
	IsSentryBusterRobot    func(client int32) bool
	IsInvulnerable         func(client int32) bool
	IsStealthed            func(client int32) bool
	IsCloakedPlayerExposed func(client int32) bool
	WorldSpaceCenter       func(entity int32) [3]float32
	IsMiniBoss             func(client int32) bool
	IsPlayerInCondition    func(client int32, condition Condition) bool
	PlayerClass            func(client int32) Class
	PlayerTeam             func(client int32) Team
	EnemyTeam              func(team Team) Team
	FindEntityByClassname  func(start int32, classname string) int32
	ObjectType             func(entity int32) Object
	EntityTeamNumber       func(entity int32) int32
	IsPlacing              func(entity int32) bool
	IsCarried              func(entity int32) bool
	HasSapper              func(entity int32) bool
	AbsOrigin              func(entity int32) [3]float32
	IsPlayer               func(entity int32) bool
	NumHealers             func(client int32) int32
	PlayerHealer           func(client int32, index int32) int32
	EntPropFloat           func(entity int32, propType PropType, prop string) float32
	EntPropEnt             func(entity int32, propType PropType, prop string) int32
	ActiveWeapon           func(client int32) int32
	WeaponID               func(weapon int32) Weapon
	EntProp                func(entity int32, propType PropType, prop string) int32
	EntityFlags            func(entity int32) int32
}

var installed Calls

// Install puts a set of answers behind the calls and returns the undo, so a
// test restores what it found. It is not safe to call from two tests at once,
// which is why nothing here runs in parallel.
func Install(c Calls) func() {
	previous := installed
	installed = c
	return func() { installed = previous }
}

func missing(name string) {
	panic(fmt.Sprintf("engine: %s was called and no answer is installed; this call has meaning on a game server and none here", name))
}

// IsClientInGame says whether the slot holds a client that has entered the game.
//
//sp:native IsClientInGame
func IsClientInGame(client int32) bool {
	if installed.IsClientInGame == nil {
		missing("IsClientInGame")
	}
	return installed.IsClientInGame(client)
}

// IsPlayerAlive says whether the client is alive right now.
//
//sp:native IsPlayerAlive
func IsPlayerAlive(client int32) bool {
	if installed.IsPlayerAlive == nil {
		missing("IsPlayerAlive")
	}
	return installed.IsPlayerAlive(client)
}

// GetClientTeam is the team index the client is on.
//
//sp:native GetClientTeam
func GetClientTeam(client int32) int32 {
	if installed.GetClientTeam == nil {
		missing("GetClientTeam")
	}
	return installed.GetClientTeam(client)
}

// SDKHasAmmo is the raw SDKCall form, which internal/body/roster uses to prove
// the generator can emit one. A port calls the plugin's wrapper below.
//
//sp:sdkcall m_hHasAmmo
func SDKHasAmmo(weapon int32) bool {
	if installed.HasAmmo == nil {
		missing("HasAmmo")
	}
	return installed.HasAmmo(weapon)
}

// SDKClip1 is the raw SDKCall form, for the same reason.
//
//sp:sdkcall m_hClip1
func SDKClip1(weapon int32) int32 {
	if installed.Clip1 == nil {
		missing("Clip1")
	}
	return installed.Clip1(weapon)
}

// Origin is where the client is standing.
//
//sp:native GetClientAbsOrigin
func Origin(client int32) (origin [3]float32) {
	if installed.Origin == nil {
		missing("GetClientAbsOrigin")
	}
	return installed.Origin(client)
}

// MaxClients is the highest client slot the server has.
//
//sp:global MaxClients
func MaxClients() int32 {
	if installed.MaxClients == nil {
		missing("MaxClients")
	}
	return installed.MaxClients()
}

// VectorDistance is how far apart two points are.
//
//sp:native GetVectorDistance
func VectorDistance(a [3]float32, b [3]float32) float32 {
	if installed.VectorDistance == nil {
		missing("GetVectorDistance")
	}
	return installed.VectorDistance(a, b)
}

// IsSentryBusterRobot says whether the robot is a sentry buster, which is
// usually not a threat worth counting.
//
//sp:plugin IsSentryBusterRobot
func IsSentryBusterRobot(client int32) bool {
	if installed.IsSentryBusterRobot == nil {
		missing("IsSentryBusterRobot")
	}
	return installed.IsSentryBusterRobot(client)
}

// IsInvulnerable says whether the player cannot be hurt right now.
//
//sp:native TF2_IsInvulnerable
func IsInvulnerable(client int32) bool {
	if installed.IsInvulnerable == nil {
		missing("TF2_IsInvulnerable")
	}
	return installed.IsInvulnerable(client)
}

// IsStealthed says whether the player is cloaked.
//
//sp:native TF2_IsStealthed
func IsStealthed(client int32) bool {
	if installed.IsStealthed == nil {
		missing("TF2_IsStealthed")
	}
	return installed.IsStealthed(client)
}

// IsCloakedPlayerExposed says whether a cloaked player can be seen anyway.
//
//sp:plugin IsCloakedPlayerExposed
func IsCloakedPlayerExposed(client int32) bool {
	if installed.IsCloakedPlayerExposed == nil {
		missing("IsCloakedPlayerExposed")
	}
	return installed.IsCloakedPlayerExposed(client)
}

// EntityWorldSpaceCenter is stocklib's: it fills the array it is given.
//
//sp:native BaseEntity_WorldSpaceCenter
func EntityWorldSpaceCenter(entity int32) (centre [3]float32) {
	if installed.WorldSpaceCenter == nil {
		missing("BaseEntity_WorldSpaceCenter")
	}
	return installed.WorldSpaceCenter(entity)
}

// ClassUnknown is TFClass_Unknown, the class a slot has before it picks one.
//
//sp:global TFClass_Unknown
func ClassUnknown() Class { return 0 }

// ConditionDazed is TFCond_Dazed, which is what a stun leaves behind.
//
//sp:global TFCond_Dazed
func ConditionDazed() Condition { return 17 }

// IsMiniBoss says whether the robot is a giant.
//
//sp:native TF2_IsMiniBoss
func IsMiniBoss(client int32) bool {
	if installed.IsMiniBoss == nil {
		missing("TF2_IsMiniBoss")
	}
	return installed.IsMiniBoss(client)
}

// IsPlayerInCondition says whether the player carries the condition.
//
//sp:native TF2_IsPlayerInCondition
func IsPlayerInCondition(client int32, condition Condition) bool {
	if installed.IsPlayerInCondition == nil {
		missing("TF2_IsPlayerInCondition")
	}
	return installed.IsPlayerInCondition(client, condition)
}

// PlayerClass is the class the player is playing.
//
//sp:native TF2_GetPlayerClass
func PlayerClass(client int32) Class {
	if installed.PlayerClass == nil {
		missing("TF2_GetPlayerClass")
	}
	return installed.PlayerClass(client)
}

// PlayerTeam is the team the player is on, as a tag rather than an index.
//
//sp:native TF2_GetClientTeam
func PlayerTeam(client int32) Team {
	if installed.PlayerTeam == nil {
		missing("TF2_GetClientTeam")
	}
	return installed.PlayerTeam(client)
}

// EnemyTeam is the team fighting this one.
//
//sp:native TF2_GetEnemyTeam
func EnemyTeam(team Team) Team {
	if installed.EnemyTeam == nil {
		missing("TF2_GetEnemyTeam")
	}
	return installed.EnemyTeam(team)
}

// ClassEngineer is TFClass_Engineer.
//
//sp:global TFClass_Engineer
func ClassEngineer() Class { return 9 }

// ObjectSapper is TFObject_Sapper.
//
//sp:global TFObject_Sapper
func ObjectSapper() Object { return 3 }

// FindEntityByClassname walks the entities of a class, from start, and answers
// -1 when there are no more.
//
//sp:native FindEntityByClassname
func FindEntityByClassname(start int32, classname string) int32 {
	if installed.FindEntityByClassname == nil {
		missing("FindEntityByClassname")
	}
	return installed.FindEntityByClassname(start, classname)
}

// ObjectType is what the building is.
//
//sp:native TF2_GetObjectType
func ObjectType(entity int32) Object {
	if installed.ObjectType == nil {
		missing("TF2_GetObjectType")
	}
	return installed.ObjectType(entity)
}

// EntityTeamNumber is the team the entity belongs to.
//
//sp:native BaseEntity_GetTeamNumber
func EntityTeamNumber(entity int32) int32 {
	if installed.EntityTeamNumber == nil {
		missing("BaseEntity_GetTeamNumber")
	}
	return installed.EntityTeamNumber(entity)
}

// IsPlacing says whether the building is still a blueprint.
//
//sp:native TF2_IsPlacing
func IsPlacing(entity int32) bool {
	if installed.IsPlacing == nil {
		missing("TF2_IsPlacing")
	}
	return installed.IsPlacing(entity)
}

// IsCarried says whether an engineer is holding the building.
//
//sp:native TF2_IsCarried
func IsCarried(entity int32) bool {
	if installed.IsCarried == nil {
		missing("TF2_IsCarried")
	}
	return installed.IsCarried(entity)
}

// HasSapper says whether the building is sapped already.
//
//sp:native TF2_HasSapper
func HasSapper(entity int32) bool {
	if installed.HasSapper == nil {
		missing("TF2_HasSapper")
	}
	return installed.HasSapper(entity)
}

// EntityAbsOrigin is stocklib's: it fills the array it is given.
//
//sp:native BaseEntity_GetAbsOrigin
func EntityAbsOrigin(entity int32) (origin [3]float32) {
	if installed.AbsOrigin == nil {
		missing("BaseEntity_GetAbsOrigin")
	}
	return installed.AbsOrigin(entity)
}

// IsPlayer says whether the entity is a client rather than a building.
//
//sp:native BaseEntity_IsPlayer
func IsPlayer(entity int32) bool {
	if installed.IsPlayer == nil {
		missing("BaseEntity_IsPlayer")
	}
	return installed.IsPlayer(entity)
}

// NumHealers is how many medics are healing the player.
//
//sp:native TF2_GetNumHealers
func NumHealers(client int32) int32 {
	if installed.NumHealers == nil {
		missing("TF2_GetNumHealers")
	}
	return installed.NumHealers(client)
}

// PlayerHealer is one of the players healing this one.
//
//sp:native TF2Util_GetPlayerHealer
func PlayerHealer(client int32, index int32) int32 {
	if installed.PlayerHealer == nil {
		missing("TF2Util_GetPlayerHealer")
	}
	return installed.PlayerHealer(client, index)
}

// PropSend is Prop_Send, the networked table.
//
//sp:global Prop_Send
func PropSend() PropType { return 1 }

// ConditionSapped is TFCond_Sapped.
//
//sp:global TFCond_Sapped
func ConditionSapped() Condition { return 15 }

// ConditionBonked is TFCond_Bonked, the phase a Scout drinking Bonk is in.
//
//sp:global TFCond_Bonked
func ConditionBonked() Condition { return 16 }

// WeaponMedigun is TF_WEAPON_MEDIGUN.
//
//sp:global TF_WEAPON_MEDIGUN
func WeaponMedigun() Weapon { return 29 }

// EntPropFloat reads a float from one of the entity's property tables.
//
//sp:native GetEntPropFloat
func EntPropFloat(entity int32, propType PropType, prop string) float32 {
	if installed.EntPropFloat == nil {
		missing("GetEntPropFloat")
	}
	return installed.EntPropFloat(entity, propType, prop)
}

// EntPropEnt reads an entity handle from one of the property tables, and
// answers -1 when it holds none.
//
//sp:native GetEntPropEnt
func EntPropEnt(entity int32, propType PropType, prop string) int32 {
	if installed.EntPropEnt == nil {
		missing("GetEntPropEnt")
	}
	return installed.EntPropEnt(entity, propType, prop)
}

// ActiveWeapon is what the player is holding.
//
//sp:native BaseCombatCharacter_GetActiveWeapon
func ActiveWeapon(client int32) int32 {
	if installed.ActiveWeapon == nil {
		missing("BaseCombatCharacter_GetActiveWeapon")
	}
	return installed.ActiveWeapon(client)
}

// WeaponID is which weapon it is.
//
//sp:native TF2Util_GetWeaponID
func WeaponID(weapon int32) Weapon {
	if installed.WeaponID == nil {
		missing("TF2Util_GetWeaponID")
	}
	return installed.WeaponID(weapon)
}

// EntProp reads an integer from one of the entity's property tables.
//
//sp:native GetEntProp
func EntProp(entity int32, propType PropType, prop string) int32 {
	if installed.EntProp == nil {
		missing("GetEntProp")
	}
	return installed.EntProp(entity, propType, prop)
}

// EntityFlags is the entity's flag word, which says among other things whether
// it has landed.
//
//sp:native GetEntityFlags
func EntityFlags(entity int32) int32 {
	if installed.EntityFlags == nil {
		missing("GetEntityFlags")
	}
	return installed.EntityFlags(entity)
}

// FlagOnGround is FL_ONGROUND.
//
//sp:global FL_ONGROUND
func FlagOnGround() int32 { return 1 }

// HasAmmo is sdkcalls.sp's wrapper round m_hHasAmmo, which is what the plugin
// actually calls.
//
//sp:plugin HasAmmo
func HasAmmo(weapon int32) bool {
	if installed.HasAmmo == nil {
		missing("HasAmmo")
	}
	return installed.HasAmmo(weapon)
}
