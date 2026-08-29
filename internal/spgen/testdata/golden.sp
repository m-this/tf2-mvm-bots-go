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
		ActionSel_Flags f;
		f.MoneyToCollect = gInputs[row][GINPUTS_MoneyToCollect] != 0;
		f.InUpgradeZone = gInputs[row][GINPUTS_InUpgradeZone] != 0;
		f.ShoppedThisBreak = gInputs[row][GINPUTS_ShoppedThisBreak] != 0;
		f.MovingToFront = gInputs[row][GINPUTS_MovingToFront] != 0;
		f.UpgradesEnabled = gInputs[row][GINPUTS_UpgradesEnabled] != 0;
		f.HasUpgraded = gInputs[row][GINPUTS_HasUpgraded] != 0;
		f.UpgradeMidRound = gInputs[row][GINPUTS_UpgradeMidRound] != 0;
		f.HasSniperRifle = gInputs[row][GINPUTS_HasSniperRifle] != 0;
		f.SniperStalled = gInputs[row][GINPUTS_SniperStalled] != 0;
		f.AttackTargetFound = gInputs[row][GINPUTS_AttackTargetFound] != 0;
		f.TankTargetFound = gInputs[row][GINPUTS_TankTargetFound] != 0;
		f.GiantToMark = gInputs[row][GINPUTS_GiantToMark] != 0;
		f.NearbyMoney = gInputs[row][GINPUTS_NearbyMoney] != 0;
		f.StickyTrapPossible = gInputs[row][GINPUTS_StickyTrapPossible] != 0;

		ActionSel_RoundState state = view_as<ActionSel_RoundState>(gInputs[row][GINPUTS_State]);
		ActionSel_Class botClass = view_as<ActionSel_Class>(gInputs[row][GINPUTS_Class]);

		printnum(view_as<int>(ActionSel_Select(state, botClass, f)));
		printnum(view_as<int>(ActionSel_SelectFilled(state, botClass, f)));
	}
}
