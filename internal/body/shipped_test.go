package body_test

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/body"
	"github.com/m-this/tf2-mvm-bots-go/internal/upstream"
)

/*
	TestGeneratedBodiesMatchTheShippedOnes

The same comparison the actions get, for the plain bodies: every function a body
package claims a SourcePawn name for is read out of the file it replaces, at the
pin, and compared on its declaration and on the sequence of calls it makes.

This did not exist while util.sp was being ported, and that was a hole rather
than a decision: those packages carried a Shipped path that nothing read, so a
port could drop a branch or reorder two calls and only the compiler would have
an opinion. The actions had this from the start; the bodies now do too.

A generated name with no shipped counterpart is skipped rather than failed: a
body may add a helper the plugin never had, and internal/body/finders holds
functions from two different files.
*/
func TestGeneratedBodiesMatchTheShippedOnes(t *testing.T) {
	// No skip: the shipped text comes from the snapshot under
	// internal/upstream, so this proof runs without the plugin repository
	// and keeps running after it is archived.
	generated, err := body.Generate("../..")
	if err != nil {
		t.Fatalf("generating: %v", err)
	}

	for _, b := range body.All {
		if b.Shipped == "" {
			continue
		}

		t.Run(b.Dir, func(t *testing.T) {
			shipped, err := upstream.ReadAt(b.Rev, strings.Split(b.Shipped, "/")...)
			if err != nil {
				t.Fatalf("reading %s at %s: %v", b.Shipped, cmp.Or(b.Rev, upstream.Rev), err)
			}

			compareBody(t, string(generated[b.Out]), shipped)
		})
	}
}

/*
emittedNames are the functions a generated file declares, which is what the
comparison walks.

public as well as stock: a callback the game calls is still a port of a shipped
function, and for a while these were emitted and never compared.
*/
var emittedNames = regexp.MustCompile(`(?m)^(?:stock|public) \w+(?:\[\])? (\w+)\(`)

// dimension is what sits between one pair of brackets.
var dimension = regexp.MustCompile(`\[[^\]]*\]`)

/*
	declarationOf is the function itself, not the first mention of its name

A name appears at its call sites long before it appears at its declaration, and
taking the first match compared a call against a definition and reported nonsense.
The declaration is the one at the start of a line, after an optional stock or
static and a return type.
*/
func declarationOf(src, name string) (string, bool) {
	at := regexp.MustCompile(`(?m)^(?:stock |static |public )?\w+(?:\[\])? ` + regexp.QuoteMeta(name) + `\(`)

	loc := at.FindStringIndex(src)
	if loc == nil {
		return "", false
	}
	return callbackOf(src[loc[0]:], name)
}

/*
	shape is the declaration without the parts a port chooses

The return type, the parameter types in order, and the defaults: those are what a
caller depends on. What is dropped is the stock the generator adds and the shipped
file often leaves off, and the parameter names, which are the port's to pick the
way the Go names them. A wrong name is a worse comment; a wrong type is a wrong
call.
*/
func shape(fn string) string {
	/* The by-reference & belongs to neither side

	The plugin writes it both ways in the same file: CKnownEntity& knownEntity
	and float &length. SourcePawn reads the two identically, so putting it on
	the type before comparing keeps a formatting choice out of the proof. */
	fn = strings.ReplaceAll(fn, " &", "& ")

	decl := strings.TrimPrefix(declOf(fn), "stock ")
	decl = strings.TrimPrefix(decl, "static ")
	decl = strings.TrimPrefix(decl, "public ")

	open := strings.Index(decl, "(")
	if open < 0 {
		return decl
	}

	head := decl[:open]
	if space := strings.LastIndexByte(head, ' '); space >= 0 {
		head = head[:space] // the return type, without the name
	}

	var params []string
	for _, p := range strings.Split(strings.Trim(decl[open+1:], "()"), ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		value := ""
		if name, def, has := strings.Cut(p, "="); has {
			p, value = strings.TrimSpace(name), " = "+strings.TrimSpace(def)
		}

		/* const is compared, because it is part of the contract

		A const array cannot be handed to a native that declares its
		parameter writable, and a caller holding one cannot pass it to a
		function that does not promise. So neither always-const nor
		never-const compiles across the plugin: the port says which with
		//sp:const and this is what checks it against the shipped
		declaration. */

		/* The type is everything but the name

		Two shapes: char[] name, where the dimensions belong to the type
		and come before the name, and float name[3], where they come
		after it. Taking the last word as the name and keeping whatever
		brackets were attached to it handles both.
		*/
		dims := ""
		if at := strings.Index(p, "["); at >= 0 && at > strings.LastIndexByte(p, ' ') {
			dims, p = p[at:], strings.TrimSpace(p[:at])
		}

		/* A parameter's dimension is documentation, so only the rank counts

		SourcePawn hands an array parameter to the callee by reference and
		tells it nothing about the length, so char[] and
		char[PLATFORM_MAX_PATH] are the same contract; sizeof is the one
		thing that would tell them apart, and
		TestNoShippedFunctionSizesItsOwnParameter checks that no shipped
		function does that. What is kept is the number of brackets, which
		is the rank and is part of the type.

		The one place the spelling does matter is a callback compared
		against a typedef: AddNormalSoundHook refuses a hook declared
		int clients[101] where NormalSHook says int clients[MAXPLAYERS].
		//sp:dim writes it out, and internal/plugin's whole-plugin
		compile is what catches a missing one. */
		dims = dimension.ReplaceAllString(dims, "[]")
		if space := strings.LastIndexByte(p, ' '); space >= 0 {
			p = p[:space]
		}

		params = append(params, p+dims+value)
	}

	return head + "(" + strings.Join(params, ", ") + ")"
}

/*
	reshaped are the functions whose declaration the port changed on purpose

One entry, and it needs the reason beside it. NestSpotFromList took the running
best spot and distance by reference and updated them in place. A generated
function zeroes its out-parameters at entry, so passing one variable as both the
candidate and the answer read the candidate as zeros: the nest was scored from the
map origin. The port takes the candidate and returns the answer separately, and
the emitter refuses the aliased shape outright now.
*/
var reshaped = map[string]string{
	"NestSpotFromList":                       "the candidate and the answer cannot be the same variable; see the aliasing refusal in internal/spbody",
	"ResetNextBot":                           "the flat list of every per-client field became one reset per package that owns some. Checked by hand against the shipped body at the pin: 26 fields, all 26 cleared, every one to the same value. A behaviour now brings its own reset or has none to bring, rather than depending on somebody remembering this list",
	"ShowWeaponPreferenceItemListMenu":       "twenty seven blocks, one per class and slot, each the same four lines around a different pool, became one loop over the pool the loadouts chain already knows how to reach. Checked by hand: WeaponPoolCount and WeaponPoolAt are generated from the same pool list GetRandomWeaponForClass uses, the spy's slots keep the shipped one-along filing, and an item the schema cannot name is still skipped",
	"CreateDisplayPanelBotPercentages":       "nine blocks of the same four lines, one per class, became one loop: the class names are a table and the shares were already indexed by class. Checked by hand: the same nine names in the same order, the same skip of a class with no share",
	"CollectUpgrades":                        "the candidate list was a JSONArray of JSONObjects, one handle per row and another per read; it is one ArrayList of five-cell rows. Checked by hand: the same slots are pushed in the same order for the same classes, the same two UI groups are skipped, the same attribute test gates the row, and the five cells hold what the five keys held",
	"ShowCurrentBotClassChances":             "nine hand-written if (classFlags & PREF_FL_X) classChoiceCount[n]++ became one loop over PrefFlagOf(c) = 1 << c. Checked by hand against the shipped file: the nine flags are 1, 2, 4, ... 256 and the nine indices are 0 through 8, in that order, so class c is counted by bit c either way. The rest of the function is unchanged",
	"DefenderBot_TouchPost":                  "one branch of the shipped one. The whole body is #if defined TFBOT_CUSTOM_SPY_CONTACT, and that define is commented out at tf2_defenderbots.sp line 33 and written nowhere else, so what compiles is the #else: the single TFBot_NoticeThreat call this port carries. The data-timer branch has never run here; Timer_RealizeSpy itself is live, called from SoundHook_General",
	"OnClientPutInServer":                    "two calls apart from the shipped one, both already settled. m_flLastCommandTime[client] = GetGameTime() is now Go_ResetCommandThrottle(client), which generated/humans.sp shows is that one assignment and nothing else. BotAim(client).Reset() is under #if defined IDLEBOT_AIMING, commented out at tf2_defenderbots.sp line 39 and written nowhere else, so it has never run. Every other line was checked field by field against the snapshot and writes the same value",
	"OnPlayerRunCmd":                         "the branches of the shipped one that compile, and only those. Its 300 lines hold five preprocessor blocks and three of the guards are written nowhere: IDLEBOT_AIMING and TESTING_ONLY are commented out at tf2_defenderbots.sp lines 39 and 23, so the reload and alt-fire holds, the BotAim aim-and-fire pair, the second BotAim upkeep and the air-control block have never run, and the minigun case under #if !defined IDLEBOT_AIMING and the whole #else arm of the aim ladder are what do. EXTRA_PLUGINBOT and MOD_ROLL_THE_DICE_REVAMPED are defined at lines 35 and 27, so PluginBot_SimulateFrame and the dice roll are ported. Checked block by block against the snapshot: every line that compiles is here, in order, and nothing else is",
	"Command_DumpUpgrades":                   "four calls renamed and nothing else: the shipped body reaches the game through CMannVsMachineUpgradeManager and CMannVsMachineUpgrades, methodmaps on an Address that a generated body has no form for, so it goes through the one-line wrappers in tf_upgrades.sp instead. manager.Address != Address_Null is IsUpgradeManagerUp, manager.Count() is UpgradeCountRaw, manager.GetUpgradeByIndex(i).Address is UpgradeAddressByIndex and upgrade.m_szAttribute() is UpgradeAttributeOf. Each wrapper is that one expression and nothing more; the call order, the two guards and both printed lines are unchanged. UpgradeCountRaw and not UpgradeCount, because UpgradeCount clamps a count this command exists to report",
	"Timer_PlayerSpawn":                      "one call short of the shipped one: VS_AddBotAttribute(data, CTFBot_IGNORE_ENEMIES) sits under #if defined IDLEBOT_AIMING, which is defined nowhere in the include tree, so it never ran. The snapshot is the text, guards and all; what compiled is what the port carries",
	"CTFBotUpgrade_Update":                   "the chooser returns a row index rather than a JSONObject, so the caller reads a cell instead of a key and frees nothing. Same interval, same window test, same refusal path, same medic beam; see CollectUpgrades for the shape",
	"CTFBotPurchaseUpgrades_ChooseUpgrade":   "it returned a JSONObject the caller then had to free; it returns the row index, and -1 where the shipped one returned null. Same walk, same seven tests in the same order, same first row that passes them all",
	"CTFBotPurchaseUpgrades_PurchaseUpgrade": "it took the JSONObject and read three keys; it takes the row and reads three cells. Same cost, same tier cap, same count arithmetic, same refusal test on the credits not moving",
	"GetUpgradePriority":                     "it took a JSONObject and read three keys out of it; it takes the three values. Same three, same order, and the caller has them in hand because it just wrote them into the row",
	"SortUpgradesByPriority":                 "the shipped sort built a second list of (priority, index), sorted that, and rebuilt a JSONArray from it. With the candidates in one block list the sort is SortCustom over them and the second list is gone. Same comparator, same order",
}

/*
	frees is how many handles a body releases

Distinct handles rather than statements, and both spellings. SourceMod frees one
with delete or with Close and the shipped files use each in different functions.
The generator writes delete from a defer, which puts the free at every way out
rather than at the one the author remembered, so one handle can be freed in two
places and is still freed once per path. What has to match is how many handles
are released, not how many times the text says so.
*/
func frees(fn string) int {
	names := map[string]bool{}

	for _, m := range freeDelete.FindAllStringSubmatch(fn, -1) {
		names[m[1]] = true
	}
	for _, m := range freeClose.FindAllStringSubmatch(fn, -1) {
		names[m[1]] = true
	}
	return len(names)
}

var (
	freeDelete = regexp.MustCompile(`delete (\w+)`)
	freeClose  = regexp.MustCompile(`(\w+)\.Close\(\)`)
)

// withoutCloses drops the Close calls, which are counted rather than sequenced.
func withoutCloses(calls []string) []string {
	out := calls[:0:0]
	for _, c := range calls {
		if c == "Close" {
			continue
		}
		out = append(out, c)
	}
	return out
}

func compareBody(t *testing.T, got, shipped string) {
	t.Helper()

	compared := 0

	for _, m := range emittedNames.FindAllStringSubmatch(got, -1) {
		name := m[1]

		if reshaped[name] != "" {
			/* A port that deliberately changed the shape

			Nothing about it is compared, which is the point: the
			shipped declaration and the shipped call sequence are
			what the port decided not to keep. The reason beside the
			name is the whole of the argument, so it has to say what
			was checked by hand instead. */
			continue
		}

		want, ok := declarationOf(shipped, name)
		if !ok {
			// A helper this port added, or a function from another
			// file: neither is a difference in what was replaced.
			continue
		}
		have, ok := declarationOf(got, name)
		if !ok {
			t.Fatalf("the generated file declares %s and then does not hold it", name)
		}

		compared++

		t.Run(name, func(t *testing.T) {
			if wantDecl, haveDecl := shape(want), shape(have); wantDecl != haveDecl {
				t.Errorf("the declaration differs:\nshipped:   %s\ngenerated: %s", wantDecl, haveDecl)
			}

			/* delete and Close are the same operation on a handle

			The shipped files write handle.Close() and the generator
			writes delete handle, because a defer puts the free at
			every way out rather than at the one the author
			remembered. Both are compared: the frees have to match in
			number, and the rest of the sequence has to match in
			order. */
			wantFrees, haveFrees := frees(want), frees(have)
			if wantFrees != haveFrees {
				t.Errorf("the body frees %d handles and the shipped one frees %d", haveFrees, wantFrees)
			}

			if wantCalls, haveCalls := withoutCloses(callsIn(want)), withoutCloses(callsIn(have)); !slices.Equal(wantCalls, haveCalls) {
				t.Errorf("the body calls a different sequence:\nshipped:   %v\ngenerated: %v", wantCalls, haveCalls)
			}
		})
	}

	if compared == 0 {
		t.Errorf("nothing was compared, so this proves nothing")
	}
}
