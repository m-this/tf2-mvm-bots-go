package navmesh

import "testing"

func TestParseMapConfig(t *testing.T) {
	const good = `
"MapConfig"
{
	// a comment
	"MovingNests"	"1"
	"Composition"	"scout, soldier,engineer"

	"EngineerNest"
	{
		"1"
		{
			"origin" "214 -1319 204"
			"zone" "outside"
		}
		"2"
		{
			"origin" "-152 -3319 -63.5"
		}
	}
}
`

	c, err := ParseMapConfig(good)
	if err != nil {
		t.Fatal(err)
	}
	if !c.MovingNests {
		t.Error("MovingNests not read")
	}
	if got, want := len(c.Composition), 3; got != want {
		t.Errorf("composition has %d classes, want %d", got, want)
	}

	nests := c.SpotsOf(EngineerNest)
	if len(nests) != 2 {
		t.Fatalf("%d nests, want 2", len(nests))
	}
	if want := (Vec3{214, -1319, 204}); nests[0].Origin != want {
		t.Errorf("nest 1 origin %v, want %v", nests[0].Origin, want)
	}
	if nests[0].Zone != "outside" {
		t.Errorf("nest 1 zone %q, want outside", nests[0].Zone)
	}
	if want := (Vec3{-152, -3319, -63.5}); nests[1].Origin != want {
		t.Errorf("nest 2 origin %v, want %v", nests[1].Origin, want)
	}
}

// TestParseMapConfigRefusals is the negative space. A config read as empty is a
// map with no spots and a check that passes for the wrong reason, so every one
// of these has to be an error rather than a quiet nothing.
func TestParseMapConfigRefusals(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"empty", ""},
		{"wrong root", `"Something" { }`},
		{"unclosed section", `"MapConfig" { "EngineerNest" { `},
		{"unterminated string", `"MapConfig" { "EngineerNest`},
		{"unquoted token", `MapConfig { }`},
		{"spot with no origin", `"MapConfig" { "EngineerNest" { "1" { "zone" "a" } } }`},
		{"origin of two numbers", `"MapConfig" { "EngineerNest" { "1" { "origin" "1 2" } } }`},
		{"origin that is not numbers", `"MapConfig" { "EngineerNest" { "1" { "origin" "a b c" } } }`},
		{"unknown top level key", `"MapConfig" { "Nonsense" "1" }`},
		{"unknown spot field", `"MapConfig" { "EngineerNest" { "1" { "origin" "1 2 3" "colour" "red" } } }`},
		{"trailing content", `"MapConfig" { } "MapConfig" { }`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := ParseMapConfig(c.text)
			if err == nil {
				t.Fatalf("parsed into %+v, want an error", cfg)
			}
		})
	}
}

// TestShippedConfigsParse reads every config the mod ships. A new one that this
// reader does not understand fails here rather than being silently skipped by
// the checks below it.
func TestShippedConfigsParse(t *testing.T) {
	for _, c := range loadConfigs(t) {
		t.Run(c.Map, func(t *testing.T) {
			if len(c.Spots) == 0 {
				t.Error("no spots")
			}
			for _, s := range c.Spots {
				if s.Index == "" {
					t.Errorf("%s has no index", s.Kind)
				}
			}
		})
	}
}
