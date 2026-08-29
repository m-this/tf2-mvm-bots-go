/* The differential probe for the ported util.sp scans.

   Hand written, like roster_probe.sp, and for the same reason: it is the
   control. What is generated is scan.sp; the world and the stubs it answers
   from are the same canned world the Go side installs.

   GetVectorDistance is stubbed here because SourceMod's vector include is not
   in the standalone SourcePawn. It is a real square root, so what is compared
   is the arithmetic the port actually does. */
#include "scan_world.inc"
#include "scan.sp"

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
	/* Two ranges either side of what the world holds, and both settings of
	   the parameter the SourcePawn callers omit. */
	for (int client = 1; client <= WORLD_SLOTS; client++)
	{
		Emit(GetNearestEnemyCount(client, 200.0, false));
		Emit(GetNearestEnemyCount(client, 200.0, true));
		Emit(GetNearestEnemyCount(client, 100000.0, false));
		Emit(GetNearestEnemyCount(client, 100000.0));
		/* Exactly the squared distance between slot 1's origin and slot
		   2's centre, so the comparison at the end of the loop is tested
		   on its boundary and not only either side of it. */
		Emit(GetNearestEnemyCount(client, 5934.8125, false));

		/* FindEnemyNearestToMe, over the filters its callers switch on.
		   The last two use the defaults, which is how util.sp calls it. */
		Emit(FindEnemyNearestToMe(client, 900000.0));
		Emit(FindEnemyNearestToMe(client, 900000.0, true));
		Emit(FindEnemyNearestToMe(client, 900000.0, false, true));
		Emit(FindEnemyNearestToMe(client, 900000.0, false, false, true));
		Emit(FindEnemyNearestToMe(client, 900000.0, false, false, false, view_as<TFClassType>(2)));
		Emit(FindEnemyNearestToMe(client, 5934.8125));

		/* The two building scans, and the spy's four passes over them.
		   The first of each uses the default range its callers omit. */
		Emit(GetNearestSappableObject(client));
		Emit(GetNearestSappableObject(client, 999999.0));
		Emit(GetNearestEnemyTeleporter(client));
		Emit(GetNearestEnemyTeleporter(client, 1000.0));
		Emit(GetBestTargetForSpy(client, 900000.0));

		/* The spy's four, over the two filters the callers vary and the
		   speed check that is off unless a caller asks for it. */
		float here[3];
		here[0] = 200.0;
		Emit(GetNearestSappablePlayer(client, 900000.0));
		Emit(GetNearestSappablePlayer(client, 900000.0, true));
		Emit(GetNearestSappablePlayer(client, 900000.0, false, view_as<TFClassType>(2), 340.0));
		Emit(GetFarthestSappablePlayer(client, 900000.0));
		Emit(GetFarthestSappablePlayer(client, 900000.0, false, TFClass_Unknown, 340.0));
		Emit(GetNearestSappablePlayerHealingSomeone(client, 900000.0));
		Emit(GetNearestSappablePlayerHealingSomeone(client, 900000.0, true, TFClass_Unknown, 340.0));
		Emit(GetEnemyPlayerNearestToPosition(client, here, 900000.0));
		Emit(GetEnemyPlayerNearestToPosition(client, here, 100.0));

		/* The money, which walks the same loop over a different classname
		   and asks two questions no building answers. The first uses the
		   default range its callers omit. */
		Emit(GetNearestCurrencyPack(client));
		Emit(GetNearestCurrencyPack(client, 1000.0));
	}
}
