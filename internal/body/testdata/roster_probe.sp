/* The differential probe for the generated bodies.

   Hand written on purpose, like internal/spgen/testdata/sweep.sp: it is the
   control, so nothing in it comes out of the generator. What is generated is
   roster.sp, which is included, and roster_world.inc, which is the canned world
   the stubs answer from and is the same world the Go side installs.

   The engine is not here, so every call the body makes is a stub that records
   what it was asked and answers from the world. That is the whole proof for a
   body that calls the engine: the answer has to match, and so does the sequence
   of calls it took to get there. A body that reached the same number by asking
   different questions has not been translated faithfully.

   Output, per case: the result cells, then how many trace cells follow, then
   the trace. */
#include "roster_world.inc"
#include "roster.sp"

native void printnum(int n);

static void Emit(int result)
{
	printnum(result);
	printnum(gTraceLen);
	for (int i = 0; i < gTraceLen; i++)
		printnum(gTrace[i]);
	gTraceLen = 0;
}

public void main()
{
	Go_ResetState();

	for (int slot = 1; slot <= WORLD_SLOTS; slot++)
		Go_SetDefenderBot(slot, gDefender[slot]);

	for (int team = 0; team < 4; team++)
		Emit(Go_AliveOnTeam(WORLD_SLOTS, team));

	for (int weapon = 0; weapon <= WORLD_SLOTS; weapon++)
		Emit(Go_LoadedRounds(weapon));

	/* The touch pair and the answer it changes, in the order the engine runs
	   them: enter the touch, ask, leave the touch, ask again. The state the
	   two hooks share is what makes the second answer differ from the
	   first, so asking outside the pair as well is the whole test. */
	for (int player = 1; player <= WORLD_SLOTS; player++)
	{
		bool value;

		Emit(view_as<int>(Go_IsBotPre(player, value)));

		Go_MyTouchPre(0, player);
		Emit(view_as<int>(Go_IsBotPre(player, value)));
		printnum(view_as<int>(value));

		Go_MyTouchPost(0, player);
		Emit(view_as<int>(Go_IsBotPre(player, value)));
	}
}
