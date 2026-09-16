package stocks

import (
	"testing"

	"github.com/m-this/tf2-mvm-bots-go/internal/engine"
)

func TestSetPlayerReadyWritesTheFlagWithoutAClientCommand(t *testing.T) {
	ready := map[int32]int32{7: 0, 8: 1}
	writes := 0
	defer engine.InstallEntities(engine.EntityCalls{
		GameRulesPropAt: func(prop string, size int32, element int32) int32 {
			if prop != "m_bPlayerReady" || size != 1 {
				t.Fatalf("read %s with size %d", prop, size)
			}
			return ready[element]
		},
		SetGameRulesPropAt: func(prop string, value int32, size int32, element int32, changeState bool) {
			if prop != "m_bPlayerReady" || size != 1 || !changeState {
				t.Fatalf("write %s value=%d size=%d changeState=%t", prop, value, size, changeState)
			}
			writes++
			ready[element] = value
		},
	})()

	SetPlayerReady(7, true)
	SetPlayerReady(8, true)
	if ready[7] != 1 || ready[8] != 1 {
		t.Fatalf("ready flags = %v", ready)
	}
	if writes != 1 {
		t.Fatalf("ready flag was written %d times, want one changed client", writes)
	}
}
