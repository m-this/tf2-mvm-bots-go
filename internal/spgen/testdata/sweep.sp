/* The differential sweep: every combination the engine can produce, through
   the generated pure function and through the generated lazy table.

   Hand written on purpose. It is the control for the generator, so nothing in
   it is generated: if this file and the generated one were both emitted from
   the same Go, agreeing would prove nothing.

   Two numbers per combination, in the order the loops produce them:
     1. ActionSel_Select, the pure port
     2. the outcome the lazy table walks to, with the bits standing in for the
        predicates the plugin would ask for */
#include "actionsel.sp"

native void printnum(int n);

public void main()
{
	for (int state = 0; state < 11; state++)
	{
		for (int botClass = 0; botClass < 10; botClass++)
		{
			for (int bits = 0; bits < (1 << ActionSel_PredicateCount); bits++)
			{
				ActionSel_Flags f;
				f.MoneyToCollect = (bits & (1 << ActionSel_PredMoneyToCollect)) != 0;
				f.InUpgradeZone = (bits & (1 << ActionSel_PredInUpgradeZone)) != 0;
				f.ShoppedThisBreak = (bits & (1 << ActionSel_PredShoppedThisBreak)) != 0;
				f.MovingToFront = (bits & (1 << ActionSel_PredMovingToFront)) != 0;
				f.UpgradesEnabled = (bits & (1 << ActionSel_PredUpgradesEnabled)) != 0;
				f.HasUpgraded = (bits & (1 << ActionSel_PredHasUpgraded)) != 0;
				f.UpgradeMidRound = (bits & (1 << ActionSel_PredUpgradeMidRound)) != 0;
				f.HasSniperRifle = (bits & (1 << ActionSel_PredHasSniperRifle)) != 0;
				f.SniperStalled = (bits & (1 << ActionSel_PredSniperStalled)) != 0;
				f.AttackTargetFound = (bits & (1 << ActionSel_PredAttackTargetFound)) != 0;
				f.TankTargetFound = (bits & (1 << ActionSel_PredTankTargetFound)) != 0;
				f.GiantToMark = (bits & (1 << ActionSel_PredGiantToMark)) != 0;
				f.NearbyMoney = (bits & (1 << ActionSel_PredNearbyMoney)) != 0;
				f.StickyTrapPossible = (bits & (1 << ActionSel_PredStickyTrapPossible)) != 0;

				if (!ActionSel_Reachable(view_as<ActionSel_Class>(botClass), f))
					continue;

				printnum(view_as<int>(ActionSel_Select(view_as<ActionSel_RoundState>(state), view_as<ActionSel_Class>(botClass), f)));

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
