/* The differential sweep: every combination the engine can produce, walked
   through the generated table.

   Hand written on purpose. It is the control for the generator, so nothing in
   it is generated: if this file and the generated one were both emitted from
   the same Go, agreeing would prove nothing. That includes the reachability
   rule and the walk, which are written out here rather than included.

   The walk below is the one the plugin runs, with the bits standing in for the
   predicates the plugin would ask for. One number per combination: the outcome
   the table walks to. */
#include "actionsel.sp"

native void printnum(int n);

/* actionsel.Reachable, written out: both sniper rifle and sniper stall are
   sniper state, so no other class can carry them. */
static bool Reachable(int botClass, int bits)
{
	if (botClass == view_as<int>(ActionSel_ClassSniper))
		return true;

	return (bits & ((1 << ActionSel_PredHasSniperRifle) | (1 << ActionSel_PredSniperStalled))) == 0;
}

public void main()
{
	for (int state = 0; state < 11; state++)
	{
		for (int botClass = 0; botClass < 10; botClass++)
		{
			for (int bits = 0; bits < (1 << ActionSel_PredicateCount); bits++)
			{
				if (!Reachable(botClass, bits))
					continue;

				int node = ActionSel_Root[state][botClass];
				for (int step = 0; step <= ActionSel_PredicateCount; step++)
				{
					if (ActionSel_NodePredicate[node] == -1)
						break;

					if ((bits & (1 << ActionSel_NodePredicate[node])) != 0)
						node = ActionSel_NodeWhenTrue[node];
					else
						node = ActionSel_NodeWhenFalse[node];
				}
				printnum(ActionSel_NodeWhenTrue[node]);
			}
		}
	}
}
