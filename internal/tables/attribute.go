package tables

import "strings"

/*
	The attribute names the upgrade ranking dispatches on

behavior/upgrade.sp ranks an upgrade by comparing its attribute name against
roughly forty string literals, 94 StrEqual sites over upgrade.sp:574-760. The
subset has no strings, so the ranking cannot move to Go as written, and the
names are duplicated between the ranking and the schema either way.

An id fixes both. The edge looks the name up once and the body switches on an
int32, which is what the ranking wanted to be.

The ids are written down, never counted. A name inserted in the middle of this
slice must not renumber the ones below it: that is the features.sp bug, and an
id that moves silently re-ranks somebody else\'s upgrade.
*/

// Attribute is one upgrade attribute name and the id the edge maps it to.
type Attribute struct {
	ID   int32
	Name string
}

// Enum is the SourcePawn constant for this attribute. The schema names carry
// spaces and inconsistent case, "SRifle Charge rate increased" beside "damage
// bonus", so the identifier is derived rather than written twice.
func (a Attribute) Enum() string {
	var b strings.Builder
	b.WriteString("ATTRIBUTE_")
	for _, r := range a.Name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

// GoIdent is the Go constant for this attribute, from the same characters the
// SourcePawn one uses so the two read as the same fact.
func (a Attribute) GoIdent() string {
	var b strings.Builder
	b.WriteString("Attr")
	upper := true
	for _, r := range a.Name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			if upper && r >= 'a' && r <= 'z' {
				r = r - 'a' + 'A'
			}
			b.WriteRune(r)
			upper = false
		default:
			upper = true
		}
	}
	return b.String()
}

// Attributes is every name behavior/upgrade.sp compares against. Adding one
// takes the next unused id; an id is never reused, because a saved ranking or a
// recorded run would read the old meaning.
var Attributes = []Attribute{
	{ID: 1, Name: "attack projectiles"},
	{ID: 2, Name: "generate rage on heal"},
	{ID: 3, Name: "weapon burn dmg increased"},
	{ID: 4, Name: "weapon burn time increased"},
	{ID: 5, Name: "maxammo metal increased"},
	{ID: 6, Name: "metal regen"},
	{ID: 7, Name: "ubercharge rate bonus"},
	{ID: 8, Name: "healing mastery"},
	{ID: 9, Name: "damage bonus"},
	{ID: 10, Name: "SRifle Charge rate increased"},
	{ID: 11, Name: "Projectile speed increased"},
	{ID: 12, Name: "fire rate bonus"},
	{ID: 13, Name: "clip size upgrade atomic"},
	{ID: 14, Name: "clip size bonus upgrade"},
	{ID: 15, Name: "faster reload rate"},
	{ID: 16, Name: "maxammo primary increased"},
	{ID: 17, Name: "maxammo secondary increased"},
	{ID: 18, Name: "engy dispenser radius increased"},
	{ID: 19, Name: "engy sentry fire rate increased"},
	{ID: 20, Name: "engy disposable sentries"},
	{ID: 21, Name: "engy building health bonus"},
	{ID: 22, Name: "melee attack rate bonus"},
	{ID: 23, Name: "uber duration bonus"},
	{ID: 24, Name: "overheal expert"},
	{ID: 25, Name: "explosive sniper shot"},
	{ID: 26, Name: "armor piercing"},
	{ID: 27, Name: "robo sapper"},
	{ID: 28, Name: "rocket specialist"},
	{ID: 29, Name: "heal on kill"},
	{ID: 30, Name: "applies snare effect"},
	{ID: 31, Name: "mad milk syringes"},
	{ID: 32, Name: "move speed bonus"},
	{ID: 33, Name: "projectile penetration"},
	{ID: 34, Name: "projectile penetration heavy"},
	{ID: 35, Name: "critboost on kill"},
	{ID: 36, Name: "mark for death"},
	{ID: 37, Name: "increase buff duration"},
	{ID: 38, Name: "effect bar recharge rate increased"},
	{ID: 39, Name: "charge recharge rate increased"},
	{ID: 40, Name: "generate rage on damage"},
	{ID: 41, Name: "bleeding duration"},
	{ID: 42, Name: "dmg taken from blast reduced"},
	{ID: 43, Name: "dmg taken from bullets reduced"},
	{ID: 44, Name: "dmg taken from fire reduced"},
	{ID: 45, Name: "health regen"},
	{ID: 46, Name: "dmg taken from crit reduced"},
	{ID: 47, Name: "damage force reduction"},
	{ID: 48, Name: "increased jump height"},
}
