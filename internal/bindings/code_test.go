package bindings

import "testing"

func TestCode(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{
			name: "line comment goes",
			src:  "a(); // Foo(b)\nc();",
			want: "a();          \nc();",
		},
		{
			name: "block comment goes",
			src:  "a(); /* Foo(b)\n Bar(c) */ d();",
			want: "a();          \n           d();",
		},
		{
			name: "string body goes, call around it stays",
			src:  `Run(x, "self.DelayedThreatNotice(%d)");`,
			want: `Run(x,                               );`,
		},
		{
			name: "character literal goes",
			src:  "if (c == '(') {}",
			want: "if (c ==    ) {}",
		},
		{
			name: "plain code is untouched",
			src:  "int x = GetClientTeam(client);",
			want: "int x = GetClientTeam(client);",
		},
		{
			name: "directive stays, its trailing comment goes",
			src:  "#define A 1 // Foo(b)\n",
			want: "#define A 1          \n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(Code([]byte(tc.src))); got != tc.want {
				t.Errorf("Code() =\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
}
