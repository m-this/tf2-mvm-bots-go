package bindings

import (
	"slices"
	"strings"
	"testing"
)

func TestParseNative(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantSig  string
		wantDefs string
	}{
		{
			name:    "plain",
			src:     `native int TF2Util_TakeHealth(int client, float flHealth);`,
			wantSig: "int TF2Util_TakeHealth(int client, float flHealth)",
		},
		{
			name:     "default argument",
			src:      `native int F(int a, int bits = 0);`,
			wantSig:  "int F(int a, int bits)",
			wantDefs: "bits=0",
		},
		{
			name:     "tagged default argument",
			src:      "native int F(int c,\n\t\tTFClassType cls = TFClass_Unknown);",
			wantSig:  "int F(int c, TFClassType cls)",
			wantDefs: "cls=TFClass_Unknown",
		},
		{
			name:    "const char array",
			src:     `native bool F(const char[] name, char[] buffer, int maxlen);`,
			wantSig: "bool F(const char[] name, char[] buffer, int maxlen)",
		},
		{
			name:    "by reference",
			src:     `native void F(float &out, int &count);`,
			wantSig: "void F(float& out, int& count)",
		},
		{
			name:    "postfix array dimensions",
			src:     `native void ConcatTransforms(const float in1[3][4], float outMat[3][4]);`,
			wantSig: "void ConcatTransforms(const float[3][4] in1, float[3][4] outMat)",
		},
		{
			name:     "untagged parameter",
			src:      `stock void F(int a, isSupport = false) { }`,
			wantSig:  "void F(int a, any isSupport)",
			wantDefs: "isSupport=false",
		},
		{
			name:    "variadic",
			src:     `native void F(const char[] fmt, any ...);`,
			wantSig: "void F(const char[] fmt, any... args)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := Parse("t.inc", []byte(tc.src))
			all := append(append([]Native{}, f.Natives...), f.Stocks...)
			if len(all) != 1 {
				t.Fatalf("got %d declarations, want 1 (refusals: %v)", len(all), f.Refusals)
			}
			if got := describe(all[0]); got != tc.wantSig {
				t.Errorf("signature = %q, want %q", got, tc.wantSig)
			}
			if got := strings.TrimPrefix(defaultsNote(all[0].Params), "// SourcePawn defaults: "); got != tc.wantDefs {
				t.Errorf("defaults = %q, want %q", got, tc.wantDefs)
			}
		})
	}
}

// describe renders a parsed declaration back into SourcePawn-ish text, so a
// test failure names the field that is wrong.
func describe(n Native) string {
	parts := make([]string, 0, len(n.Params))
	for _, p := range n.Params {
		parts = append(parts, describeType(p.Type)+ellipsis(p)+" "+p.Name)
	}
	return describeType(n.Return) + " " + n.Name + "(" + strings.Join(parts, ", ") + ")"
}

func ellipsis(p Param) string {
	if p.Variadic {
		return "..."
	}
	return ""
}

func describeType(t Type) string {
	var b strings.Builder
	if t.Const {
		b.WriteString("const ")
	}
	b.WriteString(t.Name)
	for _, d := range t.Dims {
		b.WriteString("[" + d + "]")
	}
	if t.ByRef {
		b.WriteString("&")
	}
	return b.String()
}

func TestParseMethodmap(t *testing.T) {
	const src = `
methodmap CBaseNPC < CExtNPC __nullable__
{
	public native CBaseNPC();
	public native int GetEntity();
	public static native CBaseNPC Find(int index);
	public bool IsValid() { return true; }
	property float flStepSize
	{
		public native get();
		public native set(float StepSize);
	}
	property int index
	{
		public get() { return 0; }
	}
}`
	f := Parse("t.inc", []byte(src))
	if len(f.Refusals) != 0 {
		t.Fatalf("refusals: %v", f.Refusals)
	}
	if len(f.Methodmaps) != 1 {
		t.Fatalf("got %d methodmaps, want 1", len(f.Methodmaps))
	}
	mm := f.Methodmaps[0]
	if mm.Name != "CBaseNPC" || mm.Parent != "CExtNPC" || !mm.Nullable {
		t.Errorf("header = %q < %q nullable=%v", mm.Name, mm.Parent, mm.Nullable)
	}
	wantKinds := []MethodKind{MethodConstructor, MethodPlain, MethodStatic, MethodPlain}
	if len(mm.Methods) != len(wantKinds) {
		t.Fatalf("got %d methods, want %d", len(mm.Methods), len(wantKinds))
	}
	for i, want := range wantKinds {
		if mm.Methods[i].Kind != want {
			t.Errorf("method %s kind = %v, want %v", mm.Methods[i].Name, mm.Methods[i].Kind, want)
		}
	}
	if mm.Methods[3].Native {
		t.Error("IsValid ships a body and must not be marked native")
	}
	if len(mm.Properties) != 2 {
		t.Fatalf("got %d properties, want 2", len(mm.Properties))
	}
	if p := mm.Properties[0]; !p.Get || !p.Set || !p.GetNative || !p.SetNative {
		t.Errorf("flStepSize accessors = %+v", p)
	}
	if p := mm.Properties[1]; !p.Get || p.Set || p.GetNative {
		t.Errorf("index accessors = %+v", p)
	}
}

func TestParseEnums(t *testing.T) {
	const src = `
enum TFCond
{
	TFCond_Slowed,
	TFCond_Zoomed = 4,
	TFCond_Alias = TFCond_Zoomed,
};
enum (<<= 1)
{
	FLAG_A = 1,
	FLAG_B,
	FLAG_C,
};
enum struct CEffectData
{
	float vOrigin[3];
	int iEntIndex;
	void Init() { }
}`
	f := Parse("t.inc", []byte(src))
	if len(f.Refusals) != 0 {
		t.Fatalf("refusals: %v", f.Refusals)
	}
	if len(f.Enums) != 2 || len(f.EnumStructs) != 1 {
		t.Fatalf("got %d enums and %d enum structs, want 2 and 1", len(f.Enums), len(f.EnumStructs))
	}
	if got := f.Enums[0].Entries[2].Value; got != "TFCond_Zoomed" {
		t.Errorf("alias value = %q", got)
	}
	if got := f.Enums[1].Increment; got != "<<=1" {
		t.Errorf("increment = %q", got)
	}
	es := f.EnumStructs[0]
	if len(es.Fields) != 2 || len(es.Methods) != 1 {
		t.Fatalf("enum struct = %d fields, %d methods", len(es.Fields), len(es.Methods))
	}
	if got := describeType(es.Fields[0].Type); got != "float[3]" {
		t.Errorf("vOrigin type = %q", got)
	}
}

func TestParseRefusesLoudly(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		wantReason string
	}{
		{"typeset member that is not a signature", "typeset Timer\n{\n\tint x;\n};", "not a function signature"},
		{"destructor", "methodmap Handle __nullable__ {\n\tpublic native ~Handle();\n}", "destructor"},
		{"operator overload", "stock float operator*(float a, float b) { return a; }", "operator overload"},
		{"function-like macro", "#define TEAM_ARRAY(%0, %1) (%0 + %1)", "function-like macro"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := Parse("t.inc", []byte(tc.src))
			if len(f.Refusals) != 1 {
				t.Fatalf("got %d refusals, want 1: %v", len(f.Refusals), f.Refusals)
			}
			if !strings.Contains(f.Refusals[0].Reason, tc.wantReason) {
				t.Errorf("reason = %q, want it to mention %q", f.Refusals[0].Reason, tc.wantReason)
			}
		})
	}
}

func TestParseTypeset(t *testing.T) {
	src := `
typeset Timer
{
	function Action (Handle timer);

	/* with data */
	function Action (Handle timer, any data);
};
`
	f := Parse("t.inc", []byte(src))
	if len(f.Refusals) != 0 {
		t.Fatalf("refusals: %v", f.Refusals)
	}
	if len(f.Typesets) != 1 {
		t.Fatalf("got %d typesets, want 1", len(f.Typesets))
	}
	ts := f.Typesets[0]
	if ts.Name != "Timer" || len(ts.Variants) != 2 {
		t.Fatalf("typeset = %q with %d variants, want Timer with 2", ts.Name, len(ts.Variants))
	}
	if got := ts.Variants[1].Doc; got != "with data" {
		t.Errorf("second variant doc = %q, want %q", got, "with data")
	}
	if got := len(ts.Variants[1].Params); got != 2 {
		t.Errorf("second variant has %d params, want 2", got)
	}
	if got := ts.Variants[0].Return.Name; got != "Action" {
		t.Errorf("return = %q, want Action", got)
	}
}

func TestEmitTypesetGivesEverySignatureAName(t *testing.T) {
	f := Parse("t.inc", []byte(
		"typeset Timer\n{\n\t// OnFired\n\tfunction Action (Handle timer);\n\tfunction void (Handle timer, any data);\n};"))
	out, err := Emit(f, Options{Package: "sp"})
	if err != nil {
		t.Fatalf("emitting: %v", err)
	}
	for _, want := range []string{
		"type Timer struct{ Ref int32 }",
		"type TimerOnFired func(timer Handle) Action",
		"type TimerSig",
	} {
		if !strings.Contains(string(out.Source), want) {
			t.Errorf("emitted source is missing %q:\n%s", want, out.Source)
		}
	}
}

/*
	TestInsertingASignatureRenamesNothingElse

The bug this naming exists for. Sig1 to Sig29 numbered the signatures by their
position in the include, so a signature added upstream in the middle renamed
every one after it, silently, the way a name inserted in features.sp renamed
three convars.
*/
func TestInsertingASignatureRenamesNothingElse(t *testing.T) {
	const before = "typeset Timer\n{\n\tfunction Action (Handle timer);\n\tfunction void (Handle timer, any data);\n};"
	const after = "typeset Timer\n{\n\tfunction Action (Handle timer);\n\tfunction int (Handle timer, float delay);\n\tfunction void (Handle timer, any data);\n};"

	names := func(src string) []string {
		out, err := Emit(Parse("t.inc", []byte(src)), Options{Package: "sp"})
		if err != nil {
			t.Fatalf("emitting: %v", err)
		}
		var got []string
		for line := range strings.SplitSeq(string(out.Source), "\n") {
			if name, _, ok := strings.Cut(strings.TrimPrefix(line, "type "), " func("); ok && line != name {
				got = append(got, name)
			}
		}
		return got
	}

	was, now := names(before), names(after)
	if len(now) != len(was)+1 {
		t.Fatalf("inserting one signature went from %d names to %d: %v then %v", len(was), len(now), was, now)
	}
	for _, name := range was {
		if !slices.Contains(now, name) {
			t.Errorf("%q was renamed by an insert above it: %v became %v", name, was, now)
		}
	}
}

// TestARepeatedSignatureIsOneType covers the shape two of the real typesets
// have: NativeCall declares (Handle, int) returning int twice under different
// comments. One signature is one Go type, so the repeat is dropped rather than
// emitted twice or refused.
func TestARepeatedSignatureIsOneType(t *testing.T) {
	out, err := Emit(Parse("t.inc", []byte(
		"typeset Timer\n{\n\tfunction Action (Handle timer);\n\tfunction Action (Handle timer);\n};")),
		Options{Package: "sp"})
	if err != nil {
		t.Fatalf("emitting: %v", err)
	}
	if got := strings.Count(string(out.Source), "func(timer Handle) Action"); got != 1 {
		t.Errorf("the repeated signature was emitted %d times, want 1:\n%s", got, out.Source)
	}
	if len(out.Refusals) != 0 {
		t.Errorf("a repeat is not a refusal, got %v", out.Refusals)
	}
}

func TestEmitVariadicNative(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{
			name: "trailing cells",
			src:  "native void PrintToChat(int client, const char[] format, any ...);",
			want: "func PrintToChat(client int32, format string, args ...int32)",
		},
		{
			name: "name already taken by an earlier parameter",
			src:  "native void Log(const char[] args, any ...);",
			want: "func Log(args string, args_ ...int32)",
		},
		{
			name: "no other parameters",
			src:  "native void Fail(any ...);",
			want: "func Fail(args ...int32)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Emit(Parse("t.inc", []byte(tc.src)), Options{Package: "sp"})
			if err != nil {
				t.Fatalf("emitting: %v", err)
			}
			if !strings.Contains(string(out.Source), tc.want) {
				t.Errorf("emitted source is missing %q:\n%s", tc.want, out.Source)
			}
		})
	}
}

func TestEmitRefusesADuplicateNameFromAnotherFile(t *testing.T) {
	one, err := Emit(Parse("a.inc", []byte("native int Foo();")), Options{Package: "sp"})
	if err != nil {
		t.Fatalf("emitting a.inc: %v", err)
	}
	two, err := Emit(Parse("b.inc", []byte("native int Foo();")), Options{Package: "sp", Names: one.Names})
	if err != nil {
		t.Fatalf("emitting b.inc: %v", err)
	}
	if len(two.Refusals) != 1 || !strings.Contains(two.Refusals[0].Reason, "already used") {
		t.Fatalf("refusals = %v, want the second Foo turned down", two.Refusals)
	}
	if strings.Contains(string(two.Source), "func Foo") {
		t.Error("b.inc emitted a second Foo, which is a package that does not compile")
	}
}
