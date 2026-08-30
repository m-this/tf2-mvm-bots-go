/*
Package spycheck is source/redbots3/behavior/spycheck.sp.

Spy checking, which is a thing a bot can do without being told where the Spy is.

The idea and the shape of it are Cheeseh's, from RCBot2's CSpyCheckAir and the
paranoia that starts it. It is worth saying what makes it good, because a first
attempt at this problem is always to look up the Spy's position and shoot at it,
and that is not a bot playing, it is a bot cheating quietly.

Two pieces, and neither of them reads anything a player could not see.

The first is paranoia. A team that has seen a Spy knows there is a Spy, and it
does not know where he went. So the moment one is seen, the position and the time
are remembered, and a circle around that spot grows at a walking pace. A bot
inside the circle is a bot the Spy could have reached by now, and it starts
checking. A bot outside it gets on with the wave. That is why the bots stop being
paranoid on their own, and why a Spy who works one end of the map does not put the
other end on alert.

The second is the tell, and it is the good part. A Spy is wearing a face, so
looking at the face is worthless. What gives him away is that he was not there a
moment ago. The bot takes a list of the teammates it can see, and then watches for
one that appears in view who was not in the list. A real teammate walks into view
from somewhere; a Spy tends to already be next to you.

Then it hits him. That costs nothing to be wrong about: friendly fire is off, so a
swing at a real teammate does nothing at all, and a swing at a disguised Spy hurts
him and breaks the disguise. And it stops early if the suspect fires his own
weapon, because a robot being shot at by the suspect is the alibi a Spy cannot
produce.

None of this makes a bot unstabbable, which is the point. A Spy who waits for the
check to end still gets his stab.

//sp:action DefenderSpyCheck CTFBotSpyCheck
*/
package spycheck

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Slots is the client array size, MAXPLAYERS + 1.
const Slots = 65

// How fast the circle of ground a Spy could have reached grows, in units a second
//
//sp:name SPY_PARANOIA_SPEED
const paranoiaSpeed = 320.0

// How far the paranoia reaches at most, however long ago the Spy was seen
//
//sp:name SPY_PARANOIA_RANGE_MAX
const paranoiaRangeMax = 2000.0

// How long a sighting is worth anything at all
//
//sp:name SPY_PARANOIA_MEMORY
const paranoiaMemory = 20.0

// A check is seconds out of a wave, so it is bounded at both ends
const (
	//sp:name SPY_CHECK_MIN_TIME
	checkMinTime = 2.0
	//sp:name SPY_CHECK_MAX_TIME
	checkMaxTime = 5.0
)

// Melee range, near enough. Past this the bot walks in rather than swinging at nothing
//
//sp:name SPY_CHECK_REACH
const checkReach = 80.0

// How often the bot looks for somebody who was not there before
//
//sp:name SPY_CHECK_LOOK_INTERVAL
const lookInterval = 0.1

// Knife range, and a little more, for the Spy who is simply standing behind the bot
//
//sp:name SPY_BEHIND_RANGE
const behindRange = 400.0

// How long he has to be back there. Instant would be a bot that cannot be flanked at all
//
//sp:name SPY_BEHIND_TIME
const behindTime = 0.2

/*
Where and when this team last saw a Spy

One position for the whole team, not one per bot. A Spy is seen by somebody, and a
team that has seen one talks about it. Splitting it per bot would mean the Spy has
to be seen six times before the team reacts, which is a worse model of a team than
a shared one
*/
var (
	//sp:name g_flLastSpySeenTime
	lastSeenTime float32
	//sp:name g_vLastSpySeen
	lastSeen [3]float32
)

var (
	//sp:name m_ctSpyCheckEnd
	checkEnd [Slots]float32
	//sp:name m_ctSpyCheckNextLook
	nextLook [Slots]float32
	//sp:name m_iSpyCheckSuspect
	suspectOf [Slots]int32
	//sp:name m_bSpyCheckHit
	hit [Slots]bool
	//sp:name m_bSpyCheckSeen
	seen [Slots][Slots]bool
	//sp:name m_ctSpyBehindSince
	behindSince [Slots]float32
)

// NoteSighting is a Spy seen doing something a Spy does. Everything else here
// follows from this being called.
//
//sp:name NoteSpySighting
func NoteSighting(origin [3]float32) {
	lastSeenTime = engine.GameTime()
	lastSeen = origin
}

// ResetIntel forgets the sighting.
//
//sp:name ResetSpyIntel
func ResetIntel() {
	lastSeenTime = 0.0
	lastSeen = engine.NullVector()
}

/*
IsInParanoiaRange is the ground a Spy could have covered since he was last seen.

Grows with time and stops growing at the maximum, so an old sighting eventually
means the whole area is suspect, and then the memory runs out and none of it is.
*/
//
//sp:name IsInSpyParanoiaRange
func IsInParanoiaRange(client int32) bool {
	if lastSeenTime <= 0.0 {
		return false
	}

	elapsed := engine.GameTime() - lastSeenTime

	if elapsed > paranoiaMemory {
		return false
	}

	reach := engine.MinFloat(elapsed*paranoiaSpeed, paranoiaRangeMax)

	myOrigin := engine.Origin(client)

	return engine.VectorDistance(myOrigin, lastSeen) <= reach
}

// OnStart takes the list of teammates already there and starts the clock.
func OnStart(actor int32) engine.Outcome {
	engine.PathOf(actor).SetMinLookAheadDistance(engine.DesiredPathLookAheadRange(actor))

	checkEnd[actor] = engine.GameTime() + engine.RandomFloat(checkMinTime, checkMaxTime)
	nextLook[actor] = 0.0
	suspectOf[actor] = -1
	hit[actor] = false

	// The teammates that are already there. Anybody who turns up after this is the one worth hitting
	SnapshotVisibleTeammates(actor)

	engine.SpeakConceptIfAllowed(actor, engine.ConceptCloakedSpy())

	return engine.Continue()
}

// Update watches for the one who was not there, and swings at him.
func Update(actor int32) engine.Outcome {
	if checkEnd[actor] < engine.GameTime() {
		return engine.Done("Checked for long enough")
	}

	myBot := engine.NextBotOf(actor)

	// A robot in front of the bot outranks any suspicion about a teammate behind it
	threat := myBot.Vision().PrimaryKnownThreat(true)

	if threat != 0 {
		return engine.Done("Something real to shoot at")
	}

	suspect := suspectOf[actor]

	if engine.IsValidClientIndex(suspect) && !engine.IsPlayerAlive(suspect) {
		suspect = -1
	}

	if suspect == -1 {
		if nextLook[actor] < engine.GameTime() {
			nextLook[actor] = engine.GameTime() + lookInterval

			suspect = FindTeammateWhoWasNotThere(actor)

			// Somebody turned up, so the check is worth a little longer than it had left
			if suspect != -1 {
				checkEnd[actor] = engine.GameTime() + engine.RandomFloat(checkMinTime, checkMaxTime)
			}
		}

		suspectOf[actor] = suspect
	}

	if suspect == -1 {
		return engine.Continue()
	}

	/* He is shooting at something, so he is not a Spy
	The one alibi a disguised Spy cannot produce: his weapon is a knife wearing somebody else's
	model, and firing it drops the disguise */
	if engine.TimeSinceWeaponFired(suspect) < 1.0 {
		suspectOf[actor] = -1

		return engine.Done("The suspect is fighting")
	}

	myBody := myBot.Body()

	engine.AimHeadTowards(myBody, engine.WorldSpaceCenter(suspect), engine.AimCritical(), 0.5, engine.AddressNull(), "Spy check")

	spyRange := engine.VectorDistance(engine.AbsOriginOf(actor), engine.AbsOriginOf(suspect))

	if spyRange > checkReach {
		if engine.RepathTime(actor) <= engine.GameTime() {
			engine.SetRepathTime(actor, engine.GameTime()+engine.RandomFloat(0.3, 0.4))
			engine.RepathToTarget(actor, myBot, suspect)
		}

		engine.PathOf(actor).Update(myBot)
	}

	/* Swing. Friendly fire is off, so being wrong about this costs nothing at all, and being
	right takes the disguise off him */
	if myBody.IsHeadAimingOnTarget() {
		engine.PressFireButton(actor)
	}

	if spyRange < checkReach {
		// Hit him once and move on. A bot that stands there hitting a teammate is a bot not playing
		if !hit[actor] {
			hit[actor] = true
			checkEnd[actor] = engine.GameTime() + engine.RandomFloat(0.5, 1.5)
		}
	}

	return engine.Continue()
}

// SnapshotVisibleTeammates is everybody on the bot's own team it can see right
// now.
//
//sp:name SnapshotVisibleTeammates
func SnapshotVisibleTeammates(actor int32) {
	myVision := engine.NextBotOf(actor).Vision()
	myTeam := engine.PlayerTeam(actor)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		seen[actor][i] = engine.IsClientInGame(i) && i != actor && engine.IsPlayerAlive(i) &&
			engine.PlayerTeam(i) == myTeam && myVision.IsAbleToSeeTarget(i, engine.UseFOV())
	}
}

/*
FindTeammateWhoWasNotThere is a teammate in view who was not in view when the
check started, or -1.

The whole tell. It is recorded as seen the moment it is returned, so the same
teammate is not suspected twice in one check and the bot moves on to whoever else
turns up.
*/
//
//sp:name FindTeammateWhoWasNotThere
func FindTeammateWhoWasNotThere(actor int32) int32 {
	myVision := engine.NextBotOf(actor).Vision()
	myTeam := engine.PlayerTeam(actor)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if i == actor || !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) {
			continue
		}

		if engine.PlayerTeam(i) != myTeam {
			continue
		}

		/* A human teammate is never the disguised one

		Every robot in this mode is a fake client, so a real player on RED cannot be an enemy Spy,
		and frisking him for it is noise he can see: reported from play as the team calling a player
		out as a Spy and shooting at him while he was trying to play one.

		It costs the one case where a human is on BLU through the mod's own join-blue command and
		has disguised as a defender. That is a curiosity, and being unstabbable in it is a smaller
		price than a Spy player being shot by his own team every wave. */
		if !engine.IsFakeClient(i) {
			seen[actor][i] = true
			continue
		}

		if seen[actor][i] {
			continue
		}

		if !myVision.IsAbleToSeeTarget(i, engine.UseFOV()) {
			continue
		}

		// Whoever is carrying the bomb is not a disguise, because a Spy carrying it is not disguised
		if engine.HasTheFlag(i) {
			seen[actor][i] = true
			continue
		}

		seen[actor][i] = true

		return i
	}

	return -1
}

/*
Looking behind you, which is the whole of Spy defence a bot never did

A player on a Spy wave turns round. The bot did not: it noticed a Spy only once
one had stood within SPY_BEHIND_RANGE of its back for SPY_BEHIND_TIME, which on a
wave of a hundred Spies is a description of being stabbed rather than a defence
against it.

So while the team is worried about Spies, a bot turns and looks. The glance is
short and on a loose interval: a bot that spins constantly never shoots anything,
and one that spins on a fixed beat is a bot a Spy walks in behind between beats.
*/
const (
	//sp:name SPY_GLANCE_INTERVAL_MIN
	glanceIntervalMin = 1.6
	//sp:name SPY_GLANCE_INTERVAL_MAX
	glanceIntervalMax = 3.2
	//sp:name SPY_GLANCE_TIME
	glanceTime = 0.35
	//sp:name SPY_GLANCE_RANGE
	glanceRange = 220.0
)

//sp:name m_ctNextSpyGlance
var nextGlance [Slots]float32

// UpdateGlance turns the bot round when the team is worried about Spies.
//
//sp:name UpdateSpyGlance
func UpdateGlance(client int32) {
	if !engine.Feature(engine.FeatureSpyGlance()) || !IsInParanoiaRange(client) {
		nextGlance[client] = 0.0
		return
	}

	myBot := engine.NextBotOf(client)

	// Something in front is a better use of the eyes than something that might be behind
	if myBot.Vision().PrimaryKnownThreat(true) != 0 {
		return
	}

	if nextGlance[client] > engine.GameTime() {
		return
	}

	nextGlance[client] = engine.GameTime() + engine.RandomFloat(glanceIntervalMin, glanceIntervalMax)

	myAngles := engine.ClientEyeAngles(client)
	myForward := engine.AngleForward(myAngles)

	behind := engine.EyePosition(client)
	behind[0] -= myForward[0] * glanceRange
	behind[1] -= myForward[1] * glanceRange
	behind[2] -= myForward[2] * glanceRange

	engine.AimHeadTowards(myBot.Body(), behind, engine.AimImportant(), glanceTime, engine.AddressNull(), "Checking behind me")
}

/*
UpdateIntel is what this bot can honestly claim to have seen of a Spy, fed to the
team's memory of one.

A disguised Spy is deliberately not a sighting. He is wearing a face and the bot
believes the face, which is the whole contract of the disguise and the reason the
tell above is worth having. What counts is a Spy with no disguise on, a Spy whose
cloak has been broken, and, separately, a Spy standing at the bot's back.

The last one is not RCBot2's and it is not paranoia. A player who has somebody at
knife distance behind him for half a second turns around, whatever he believes
about who it is.
*/
//
//sp:name UpdateSpyIntel
func UpdateIntel(client int32) {
	UpdateGlance(client)

	myVision := engine.NextBotOf(client).Vision()
	enemyTeam := engine.PlayerEnemyTeam(client)

	myOrigin := engine.Origin(client)
	myAngles := engine.ClientEyeAngles(client)
	myForward := engine.AngleForward(myAngles)

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if !engine.IsClientInGame(i) || !engine.IsPlayerAlive(i) {
			continue
		}

		if engine.PlayerTeam(i) != enemyTeam || engine.PlayerClass(i) != engine.ClassSpy() {
			continue
		}

		// Cloak is cloak. A bot that sees through it is a bot no Spy can play against
		if engine.IsStealthed(i) && !engine.IsCloakedPlayerExposed(i) {
			continue
		}

		theirOrigin := engine.Origin(i)

		if engine.VectorDistance(myOrigin, theirOrigin) <= behindRange {
			toThem := engine.SubtractVectors(theirOrigin, myOrigin)
			_, toThem = engine.NormalizeVector(toThem)

			// Behind, which is anything the bot is not roughly facing
			if engine.VectorDotProduct(myForward, toThem) < 0.0 {
				if behindSince[client] <= 0.0 {
					behindSince[client] = engine.GameTime()
				} else if engine.GameTime()-behindSince[client] >= behindTime {
					behindSince[client] = 0.0

					myVision.AddKnownEntity(i)
					NoteSighting(theirOrigin)
				}

				continue
			}
		}

		// Nothing pretending to be anything, in plain view. That is a sighting
		if !engine.IsPlayerInCondition(i, engine.ConditionDisguised()) && myVision.IsAbleToSeeTarget(i, engine.UseFOV()) {
			NoteSighting(theirOrigin)
		}
	}
}

// Reset forgets what this bot was checking.
//
//sp:name ResetSpyCheck
func Reset(client int32) {
	behindSince[client] = 0.0
	checkEnd[client] = 0.0
	suspectOf[client] = -1
}

// IsPossible says whether frisking the team is worth the bot's time.
//
//sp:name CTFBotSpyCheck_IsPossible
func IsPossible(client int32) bool {
	if !engine.IsPlayerAlive(client) || engine.IsInUpgradeZone(client) {
		return false
	}

	// A bot in the middle of a fight has better things to do than frisk its own team
	if engine.NextBotOf(client).Vision().PrimaryKnownThreat(true) != 0 {
		return false
	}

	/* An engineer holding a nest is doing the one job nobody else can do, and the sentry is the
	spy check: anything that walks into it while sapping is already being shot at */
	if engine.PlayerClass(client) == engine.ClassEngineer() {
		return false
	}

	return IsInParanoiaRange(client)
}
