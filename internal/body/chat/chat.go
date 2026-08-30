/*
Package chat is what is left of source/redbots3/util.sp: saying something to one
team.
*/
package chat

import "github.com/m-this/tf2-mvm-bots-go/internal/engine"

/*
PrintToChatTeam says a line to everybody on one team.

Formatted per player rather than once, because the translation target decides the
language and a server can have several: SetGlobalTransTarget is what says who the
next VFormat is for.
*/
//
//sp:name PrintToChatTeam
//
//nolint:revive // unused-parameter: the arguments are the caller's, handed to VFormat by index
func PrintToChatTeam(team int32, format string, args ...any) {
	// The shipped buffer, which decides where a long line is cut.
	var buffer engine.Line

	for i := int32(1); i <= engine.MaxClients(); i++ {
		if engine.IsClientInGame(i) && engine.GetClientTeam(i) == team {
			engine.SetGlobalTransTarget(i)

			buffer = engine.VFormat(format, 3)

			engine.PrintToChatText(i, "%s", buffer)
		}
	}
}
