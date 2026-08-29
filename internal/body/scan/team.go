package scan

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// PlayerEnemyTeam is util.sp:1044, GetPlayerEnemyTeam. Every one of the scans
// above opened by calling it, so it was an extern until they were all across;
// now it is here and the extern is gone, because a function owned in both
// places is the duplication this repository exists to remove.
//
//sp:name GetPlayerEnemyTeam
func PlayerEnemyTeam(client int32) engine.Team {
	return engine.EnemyTeam(engine.PlayerTeam(client))
}
