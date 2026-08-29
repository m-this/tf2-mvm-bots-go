package actionsel

import "testing"

// TestClosedBugs is the beads the epic names, each as the combination that
// produced it. The want column is what the plugin does today, because this is
// a port: a bead that is closed in the plugin reads as the fixed answer, and
// the one that is open reads as the defect.
func TestClosedBugs(t *testing.T) {
	tests := []struct {
		name  string
		bead  string
		state RoundState
		class Class
		flags Flags
		want  Action
	}{
		{
			name:  "an engineer who already shopped is left to his nest",
			bead:  "mvm-7kr",
			state: RoundBetweenRounds,
			class: ClassEngineer,
			flags: Flags{ShoppedThisBreak: true, UpgradesEnabled: true},
			want:  ActionKeepOwnBreakBehaviour,
		},
		{
			name:  "an engineer who has not shopped goes shopping",
			bead:  "mvm-7kr",
			state: RoundBetweenRounds,
			class: ClassEngineer,
			flags: Flags{UpgradesEnabled: true},
			want:  ActionGotoUpgradeBetweenRounds,
		},
		{
			name:  "a rifle sniper is refused the front and left to his perch",
			bead:  "mvm-pvt",
			state: RoundBetweenRounds,
			class: ClassSniper,
			flags: Flags{ShoppedThisBreak: true, UpgradesEnabled: true, HasSniperRifle: true},
			want:  ActionKeepOwnBreakBehaviour,
		},
		{
			name:  "a sniper with no rifle walks to the front",
			bead:  "mvm-pvt",
			state: RoundBetweenRounds,
			class: ClassSniper,
			flags: Flags{ShoppedThisBreak: true, UpgradesEnabled: true},
			want:  ActionMoveToFrontShoppingDone,
		},
		{
			name:  "the medic follows his patient to the front",
			bead:  "mvm-e4g",
			state: RoundBetweenRounds,
			class: ClassMedic,
			flags: Flags{ShoppedThisBreak: true, UpgradesEnabled: true},
			want:  ActionMoveToFrontShoppingDone,
		},
		{
			name:  "a stalled sniper fights like one who never had a rifle",
			bead:  "mvm-489",
			state: RoundRunning,
			class: ClassSniper,
			flags: Flags{HasUpgraded: true, UpgradesEnabled: true, HasSniperRifle: true, SniperStalled: true},
			want:  ActionDefenderAttackSniper,
		},
		{
			name:  "a sniper who kept his mission is left sniping",
			bead:  "mvm-489",
			state: RoundRunning,
			class: ClassSniper,
			flags: Flags{HasUpgraded: true, UpgradesEnabled: true, HasSniperRifle: true},
			want:  ActionKeepSnipingPosition,
		},
		{
			name:  "a stalled sniper is sent to the front in the break",
			bead:  "mvm-489",
			state: RoundBetweenRounds,
			class: ClassSniper,
			flags: Flags{ShoppedThisBreak: true, UpgradesEnabled: true, HasSniperRifle: true, SniperStalled: true},
			want:  ActionMoveToFrontShoppingDone,
		},
		{
			name:  "a scout with no money, no giant and no target is stranded, and stays stranded",
			bead:  "mvm-vnn",
			state: RoundRunning,
			class: ClassScout,
			flags: Flags{HasUpgraded: true, UpgradesEnabled: true},
			want:  ActionStrandedAsShipped,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Select(tc.state, tc.class, tc.flags); got != tc.want {
				t.Errorf("%s: Select answers %s, want %s", tc.bead, name(got), name(tc.want))
			}
		})
	}
}
