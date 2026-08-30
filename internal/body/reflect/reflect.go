/*
Package reflect is the part of source/redbots3/util.sp that names things by what
they are: a projectile an airblast can send back, and a sapper by its item
definition.
*/
package reflect

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
CanBeReflected says an airblast would send this projectile back.

By classname, because that is what the game gives a projectile and there is no
attribute that says it. The jars are matched by prefix, since every jar is one.
*/
//
//sp:name CanBeReflected
func CanBeReflected(projectile int32) bool {
	classname := engine.EntityClassname(projectile)

	if engine.StrEqualFolded(classname, "tf_projectile_arrow", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_ball_ornament", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_cleaver", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_energy_ball", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_flare", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_healing_bolt", false) ||
		engine.StrContains(classname, "tf_projectile_jar", false) != -1 ||
		engine.StrEqualFolded(classname, "tf_projectile_pipe", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_rocket", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_sentryrocket", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_stun_ball", false) ||
		engine.StrEqualFolded(classname, "tf_projectile_balloffire", false) {
		return true
	}

	return false
}

// IsItemDefIndexSapper says the item is a sapper, of which the game has seven.
//
//sp:name IsItemDefIndexSapper
func IsItemDefIndexSapper(itemDefIndex int32) bool {
	switch itemDefIndex {
	case 735, 736, 810, 831, 933, 1080, 1102:
		return true
	}

	return false
}
