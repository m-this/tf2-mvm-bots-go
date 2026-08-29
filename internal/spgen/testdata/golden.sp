/* The golden-input harness: a table of rows emitted from a Go struct, one
   result read back per row.

   Hand written, like sweep.sp, and for the same reason. Where sweep.sp walks
   the whole domain in loops, this one takes the exact rows the Go test chose,
   which is what a golden table is for: a failure names a row somebody can go
   and look at. */
#include "actionsel.sp"
#include <golden_inputs>

native void printnum(int n);

public void main()
{
	for (int row = 0; row < GINPUTS_ROWS; row++)
	{
		int bits;
		bits |= gInputs[row][GINPUTS_MoneyToCollect] << ActionSel_PredMoneyToCollect;
		bits |= gInputs[row][GINPUTS_InUpgradeZone] << ActionSel_PredInUpgradeZone;
		bits |= gInputs[row][GINPUTS_ShoppedThisBreak] << ActionSel_PredShoppedThisBreak;
		bits |= gInputs[row][GINPUTS_MovingToFront] << ActionSel_PredMovingToFront;
		bits |= gInputs[row][GINPUTS_UpgradesEnabled] << ActionSel_PredUpgradesEnabled;
		bits |= gInputs[row][GINPUTS_HasUpgraded] << ActionSel_PredHasUpgraded;
		bits |= gInputs[row][GINPUTS_UpgradeMidRound] << ActionSel_PredUpgradeMidRound;
		bits |= gInputs[row][GINPUTS_HasSniperRifle] << ActionSel_PredHasSniperRifle;
		bits |= gInputs[row][GINPUTS_SniperStalled] << ActionSel_PredSniperStalled;
		bits |= gInputs[row][GINPUTS_AttackTargetFound] << ActionSel_PredAttackTargetFound;
		bits |= gInputs[row][GINPUTS_TankTargetFound] << ActionSel_PredTankTargetFound;
		bits |= gInputs[row][GINPUTS_GiantToMark] << ActionSel_PredGiantToMark;
		bits |= gInputs[row][GINPUTS_NearbyMoney] << ActionSel_PredNearbyMoney;
		bits |= gInputs[row][GINPUTS_StickyTrapPossible] << ActionSel_PredStickyTrapPossible;

		int node = ActionSel_Root[gInputs[row][GINPUTS_State]][gInputs[row][GINPUTS_Class]];
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
