package x

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

// Ask calls an engine function this emission was not told how to write.
func Ask(client int32) bool { return engine.IsClientInGame(client) }
