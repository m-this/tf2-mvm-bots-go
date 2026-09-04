package gameevents

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

func TestDisplacedDefenderRecovery(t *testing.T) {
	tests := []struct {
		name       string
		fake       bool
		defender   bool
		disconnect bool
		oldTeam    engine.Team
		team       engine.Team
		wantKick   bool
	}{
		{name: "defender leaves RED", fake: true, defender: true, oldTeam: engine.TeamRed(), team: engine.TeamBlue(), wantKick: true},
		{name: "intentional disconnect", fake: true, defender: true, disconnect: true, oldTeam: engine.TeamRed(), team: 0},
		{name: "ordinary BLU robot", fake: true, oldTeam: engine.TeamRed(), team: engine.TeamBlue()},
		{name: "defender stays on RED", fake: true, defender: true, oldTeam: engine.TeamRed(), team: engine.TeamRed()},
		{name: "human outside RED", oldTeam: 0, team: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleared := false
			kicked := false
			defer engine.InstallRegistrations(engine.RegisterCalls{
				EventInt: func(_ engine.Event, key string) int32 {
					switch key {
					case "userid":
						return 41
					case "team":
						return int32(tt.team)
					case "oldteam":
						return int32(tt.oldTeam)
					default:
						t.Fatalf("unexpected integer event field %q", key)
						return 0
					}
				},
				EventBool: func(_ engine.Event, key string) bool {
					if key != "disconnect" {
						t.Fatalf("unexpected boolean event field %q", key)
					}
					return tt.disconnect
				},
			})()
			defer engine.InstallBluAssists(engine.BluAssistCalls{
				ClientOfUserID: func(userid int32) int32 {
					if userid != 41 {
						t.Fatalf("userid = %d, want 41", userid)
					}
					return 7
				},
			})()
			defer engine.InstallSpyChecks(engine.SpyCheckCalls{
				IsFakeClient: func(client int32) bool {
					if client != 7 {
						t.Fatalf("client = %d, want 7", client)
					}
					return tt.fake
				},
			})()
			defer engine.InstallFronts(engine.FrontCalls{
				DefenderBotFlag: func(client int32) bool {
					if client != 7 {
						t.Fatalf("client = %d, want 7", client)
					}
					return tt.defender
				},
			})()
			defer engine.InstallCompositions(engine.CompositionCalls{
				ClearBuildingsBeforeKick: func(client int32) {
					if client != 7 {
						t.Fatalf("cleared client = %d, want 7", client)
					}
					cleared = true
				},
			})()
			defer engine.InstallRosterCounts(engine.RosterCountCalls{
				KickClient: func(client int32, reason string) {
					if client != 7 {
						t.Fatalf("kicked client = %d, want 7", client)
					}
					if reason != "BotManager3: restoring the RED lineup" {
						t.Errorf("kick reason = %q", reason)
					}
					kicked = true
				},
			})()

			EventPlayerTeam(1, "player_team", true)

			if kicked != tt.wantKick {
				t.Errorf("kicked = %t, want %t", kicked, tt.wantKick)
			}
			if cleared != tt.wantKick {
				t.Errorf("buildings cleared = %t, want %t", cleared, tt.wantKick)
			}
		})
	}
}
