/*
Package pluginbot is esPluginBot out of source/redbots3/nextbot_behavior.sp:
where a plugin-driven bot is walking to.

Its own package rather than part of declarations, because a comparison reads one
shipped file and this record came from a different one.
*/
package pluginbot

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is MAXPLAYERS + 1, the client array size.
const Slots = 65

/*
Record is esPluginBot: where a plugin-driven bot is walking to.

A goal is either a place or an entity and never both, which is what the two
setters enforce between them.
*/
//
//sp:name esPluginBot
type Record struct {
	Pathing        bool       `sp:"bPathing"`
	PathGoal       [3]float32 `sp:"vecPathGoal"`
	PathGoalEntity int32      `sp:"iPathGoalEntity"`
}

// Reset forgets where it was going.
//
//sp:name Reset
func (p *Record) Reset() {
	p.Pathing = false
	p.PathGoal = engine.NullVector()
	p.PathGoalEntity = -1
}

// HasPathGoalVector says the goal is a place.
//
//sp:name HasPathGoalVector
func (p *Record) HasPathGoalVector() bool {
	return !engine.VectorIsZero(p.PathGoal)
}

// HasPathGoalEntity says the goal is an entity.
//
//sp:name HasPathGoalEntity
func (p *Record) HasPathGoalEntity() bool {
	return p.PathGoalEntity != -1
}

// SetPathGoalVector aims it at a place.
//
//sp:name SetPathGoalVector
//sp:const vec
func (p *Record) SetPathGoalVector(vec [3]float32) {
	// You can only set one or the other, not both.
	p.PathGoalEntity = -1
	p.PathGoal = vec
}

// SetPathGoalEntity aims it at an entity.
//
//sp:name SetPathGoalEntity
func (p *Record) SetPathGoalEntity(entity int32) {
	p.PathGoal = engine.NullVector()
	p.PathGoalEntity = entity
}

//sp:name g_arrPluginBot
//nolint:unused // emitted, not read from Go: the generated files that read it are SourcePawn
var pluginBot [Slots]Record
