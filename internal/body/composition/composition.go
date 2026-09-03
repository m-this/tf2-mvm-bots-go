/*
Package composition is the named team out of source/tf2_defenderbots.sp: the
lineup a server owner typed, the classes they banned, and the seats the lineup
still wants filled.
*/
package composition

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Classes is one past TFClass_Engineer, the size of a per-class count.
const Classes = 10

// Seats is MAXPLAYERS + 1, the most entries a typed lineup can hold.
const Seats = 65

/*
	GetWantedTeamComposition is the lineup to fill RED with, or an empty string

The convar wins over the map. Somebody who typed a team into the console is
answering a question the map file guessed at, and the map is a default rather
than an instruction.

The map's own answer exists because the right team is not the same on every map:
Mannworks is full of deflector Heavies, which eat a Soldier's rockets and do
nothing to a second Heavy, and Coal Town is one long bottleneck full of Spies,
where a Pyro is worth more than the reach.
*/
//
//sp:name GetWantedTeamComposition
//sp:writable out
func GetWantedTeamComposition(out engine.Text, maxlen int32) {
	engine.TeamComposition().StringInto(out, maxlen)

	if out[0] != 0 {
		return
	}

	engine.StrcopyFromText(out, maxlen, engine.MapComposition())
}

// IsClassInTeamComposition asks whether the named team wants that class
// anywhere in the team it names.
//
//sp:name IsClassInTeamComposition
//sp:default bTypedTeamOnly false
func IsClassInTeamComposition(class engine.Text, bTypedTeamOnly bool) bool {
	var list engine.Text

	if bTypedTeamOnly {
		engine.TeamComposition().StringInto(list, engine.TextSize())
	} else {
		engine.GetWantedTeamComposition(list, engine.TextSize())
	}

	if list[0] == 0 {
		return false
	}

	wanted := engine.ClassIndexFromString(class)

	if wanted == engine.ClassUnknown() {
		return false
	}

	var entries [Seats]engine.Text
	count := engine.ExplodeSeatList(list, ",", entries, Seats, engine.TextSize())

	for i := int32(0); i < count; i++ {
		engine.TrimString(entries[i])

		if engine.ClassIndexFromString(entries[i]) == wanted {
			return true
		}
	}

	return false
}

/*
	IsBotClassBlacklisted says the server was told never to play that class

A team somebody typed out is more specific than the blacklist, so what it asks
for is never blacklisted. The map config's own composition is not that: it is
this mod's guess at a good team for the map, and a guess does not get to
overrule a class the server was told never to play. Reported from a play-test as
seats set to "Let the mod pick" drawing unticked classes.
*/
//
//sp:name IsBotClassBlacklisted
func IsBotClassBlacklisted(class engine.Text) bool {
	if engine.IsClassInTeamComposition(class, true) {
		return false
	}

	var list engine.Text
	engine.ClassBlacklist().StringInto(list, engine.TextSize())

	if list[0] == 0 {
		return false
	}

	wanted := engine.ClassIndexFromString(class)

	if wanted == engine.ClassUnknown() {
		return false
	}

	var entries [Classes - 1]engine.Text
	count := engine.ExplodeClassList(list, ",", entries, Classes-1, engine.TextSize())

	for i := int32(0); i < count; i++ {
		engine.TrimString(entries[i])

		if engine.ClassIndexFromString(entries[i]) == wanted {
			return true
		}
	}

	return false
}

// PickAllowedBotClass keeps the wanted class unless it is blacklisted, and
// otherwise draws from the classes that are not.
//
//sp:name PickAllowedBotClass
//sp:writable buffer
func PickAllowedBotClass(wanted engine.Text, buffer engine.Text, maxlen int32) {
	engine.StrcopyFromText(buffer, maxlen, wanted)

	if !engine.IsBotClassBlacklisted(wanted) {
		return
	}

	var candidates [Classes - 1]engine.Text
	total := int32(0)

	for i := engine.ClassScout(); i <= engine.ClassEngineer(); i++ {
		if !engine.IsBotClassBlacklisted(engine.RawPlayerClassName(i)) {
			engine.StrcopyFromText(candidates[total], engine.ClassNameMax(), engine.RawPlayerClassName(i))
			total++
		}
	}

	// Everything is blacklisted, which cannot be meant: the list is ignored.
	if total == 0 {
		return
	}

	engine.StrcopyFromText(buffer, maxlen, candidates[engine.RandomInt(0, total-1)])
}

/*
	CollectMissingTeamComposition names the seats the lineup still wants filled

The list is what the team should look like, not what to add: every call counts
the bots already on RED against it first and names only what is missing. So a
top-up in the middle of a wave converges on the same team as the first fill,
whatever order the seats emptied in. A list shorter than the seats leaves the
rest to the lineup mode.

A seat is where a class sits in that list, counted from 1, and it comes out
alongside the class name because the loadout file can name one seat rather than
every engineer at once.
*/
//
//sp:name CollectMissingTeamComposition
func CollectMissingTeamComposition(classes engine.List, seats engine.List, count int32) int32 {
	var list engine.Text
	engine.GetWantedTeamComposition(list, engine.TextSize())

	if list[0] == 0 {
		return 0
	}

	var wanted [Seats]engine.Text
	total := engine.ExplodeSeatList(list, ",", wanted, Seats, engine.TextSize())

	// How many bots of each class already hold a seat.
	var held [Classes]int32

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && engine.IsDefenderBot(i) && engine.ClientTeam(i) == engine.TeamRed() {
			held[engine.PlayerClass(i)]++
		}
	}

	collected := int32(0)

	for i := int32(0); i < total && collected < count; i++ {
		engine.TrimString(wanted[i])

		class := engine.ClassIndexFromString(wanted[i])

		if class == engine.ClassUnknown() {
			continue
		}

		if held[class] > 0 {
			held[class]--
			continue
		}

		classes.PushStringText(wanted[i])
		seats.Push(i + 1)
		collected++
	}

	return collected
}

/*
	AddBotsFromTeamComposition fills the empty seats from the named team

Zero when the convar named nothing to add, and the caller asks the lineup mode
for the rest.
*/
//
//sp:name AddBotsFromTeamComposition
func AddBotsFromTeamComposition(count int32) int32 {
	classes := engine.NewListSized(engine.ClassNameMax())
	seats := engine.NewList()
	added := CollectMissingTeamComposition(classes, seats, count)

	for i := int32(0); i < classes.Length(); i++ {
		class := classes.GetString(i)
		engine.NoteBotSeatPending(seats.Get(i))
		engine.AddDefenderTFBotOf(1, class, "red", "expert")
	}

	classes.Close()
	seats.Close()

	if added > 0 {
		engine.PrintToChatAll("%s Adding %d bot(s)...", engine.PluginPrefix(), added)
	}

	engine.LogMessage("Fill: the named team filled %d of %d", added, count)

	return added
}

/*
	ReseatDefenderBots kicks the bots the lineup no longer asks for

Only as many as there are seats nobody holds, so a lineup that matches the team
kicks nobody. The bots are collected before the first kick and rechecked after
it, because kicking one changes who is in the game.
*/
//
//sp:name ReseatDefenderBots
func ReseatDefenderBots() int32 {
	var list engine.Text
	engine.GetWantedTeamComposition(list, engine.TextSize())

	if list[0] == 0 {
		return 0
	}

	var wanted [Seats]engine.Text
	total := engine.ExplodeSeatList(list, ",", wanted, Seats, engine.TextSize())

	// Bots of a class the list no longer asks for, and the clients holding them.
	var spare [Classes]int32
	bots := engine.NewList()

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && engine.IsDefenderBot(i) && engine.ClientTeam(i) == engine.TeamRed() {
			spare[engine.PlayerClass(i)]++
			bots.Push(i)
		}
	}

	// Seats the list names that nobody holds, which is what there is room to kick.
	missing := int32(0)

	for i := int32(0); i < total; i++ {
		engine.TrimString(wanted[i])

		class := engine.ClassIndexFromString(wanted[i])

		// A blank or a typo leaves the seat to the lineup mode, so it asks
		// for nobody in particular.
		if class == engine.ClassUnknown() {
			continue
		}

		if spare[class] > 0 {
			spare[class]--
		} else {
			missing++
		}
	}

	kicked := int32(0)

	for i := int32(0); i < bots.Length() && kicked < missing; i++ {
		client := bots.Get(i)

		// Rechecked rather than trusted: the list was taken before the first kick.
		if !engine.IsClientInGame(client) {
			continue
		}

		class := engine.PlayerClass(client)

		if spare[class] < 1 {
			continue
		}

		spare[class]--
		kicked++
		engine.ClearBuildingsBeforeKick(client)
		engine.KickClient(client, "BotManager3: the lineup changed")
	}

	bots.Close()

	if kicked > 0 {
		engine.LogMessage("Reseat: the lineup wants %d seat(s) nobody holds, kicked %d bot(s) for them", missing, kicked)
		engine.PrintToChatAll("%s Changing %d bot(s) to match the new lineup...", engine.PluginPrefix(), kicked)
	}

	return kicked
}

/*
	AddBotsWithPresetTeamComp seats one of the three preset lineups

Dead: nothing calls it and nothing else reads the table it draws from. Ported
rather than deleted, because mvm-z83.41 says a port does not delete what it does
not understand, and mvm-z83.80 is where that question belongs.
*/
//
//sp:name AddBotsWithPresetTeamComp
//sp:default count 6
//sp:default teamType 0
func AddBotsWithPresetTeamComp(count int32, teamType int32) {
	total := int32(0)

	for i := int32(0); i < count; i++ {
		// We are done here.
		if total >= count {
			break
		}

		// More was asked for than the lineup holds, so cycle back to its start.
		if i >= engine.PresetLineupSeats() {
			i = 0
		}

		engine.AddDefenderTFBotOf(1, engine.BotTeamComposition(teamType, i), "red", "expert")
		total++
	}
}
