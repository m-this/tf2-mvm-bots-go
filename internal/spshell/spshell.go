package spshell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrNoToolchain says spcomp, spshell or the include tree was not configured.
var ErrNoToolchain = errors.New("spshell: no toolchain")

// Toolchain is the standalone SourcePawn build, all three parts of it.
type Toolchain struct {
	Spcomp     string
	Spshell    string
	IncludeDir string
}

// ToolchainFromEnv reads SPCOMP, SPSHELL and SPINCLUDE.
func ToolchainFromEnv() (Toolchain, error) {
	t := Toolchain{
		Spcomp:     os.Getenv("SPCOMP"),
		Spshell:    os.Getenv("SPSHELL"),
		IncludeDir: os.Getenv("SPINCLUDE"),
	}
	for name, path := range map[string]string{"SPCOMP": t.Spcomp, "SPSHELL": t.Spshell, "SPINCLUDE": t.IncludeDir} {
		if path == "" {
			return Toolchain{}, fmt.Errorf("%w: %s is unset", ErrNoToolchain, name)
		}
		if _, err := os.Stat(path); err != nil {
			return Toolchain{}, fmt.Errorf("%w: %s: %w", ErrNoToolchain, name, err)
		}
	}
	return t, nil
}

// Run compiles sourcePath and returns the cells the plugin printed, one per
// printnum call, in order.
//
// includes are written into a directory put ahead of the SourcePawn include
// tree, so a generated golden table or a generated body is included by name
// without anything being left on disk afterwards.
func (t Toolchain) Run(ctx context.Context, sourcePath string, includes map[string]string) ([]int32, error) {
	dir, err := os.MkdirTemp("", "spshell")
	if err != nil {
		return nil, fmt.Errorf("temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	smx := filepath.Join(dir, "golden.smx")
	if err := t.build(ctx, sourcePath, smx, dir, includes); err != nil {
		return nil, err
	}
	return t.run(ctx, smx)
}

// build writes the injected includes and compiles, which is everything Run and
// Compile share.
func (t Toolchain) build(ctx context.Context, sourcePath, smx, dir string, includes map[string]string) error {
	for name, body := range includes {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}
	return t.compile(ctx, sourcePath, smx, dir)
}

/*
	Compile builds sourcePath and throws the plugin away

For the generated edge, which calls into the engine and so cannot run anywhere
but a game server. Compiling is the whole check there: a reserved word used as a
parameter name, a typo in a behaviour name and a switch over an outcome the enum
does not have are all compile errors.
*/
func (t Toolchain) Compile(ctx context.Context, sourcePath string, includes map[string]string) error {
	dir, err := os.MkdirTemp("", "spshell")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	return t.build(ctx, sourcePath, filepath.Join(dir, "out.smx"), dir, includes)
}

// Triple is one golden input: the three floats a decision function takes.
type Triple [3]float32

// ScoreTriples compiles sourcePath against inputs and returns one float32 per
// input, as the plugin's main printed them.
//
// The plugin reads a gInputs array from a generated golden_inputs.inc and prints
// each result with printnum(view_as<int>(score)), so what comes back is the
// exact bit pattern rather than a rounded printf.
func (t Toolchain) ScoreTriples(ctx context.Context, sourcePath string, inputs []Triple) ([]float32, error) {
	cells, err := t.Run(ctx, sourcePath, map[string]string{"golden_inputs.inc": inputsInclude(inputs)})
	if err != nil {
		return nil, err
	}
	scores := make([]float32, 0, len(cells))
	for _, c := range cells {
		// The plugin prints view_as<int>(score), so this reinterprets the
		// cell's bits rather than converting a number, which is the point.
		scores = append(scores, math.Float32frombits(uint32(c))) //nolint:gosec // G115: a cell is 32 bits either way
	}
	return scores, nil
}

func (t Toolchain) compile(ctx context.Context, sourcePath, smx, includeDir string) error {
	cmd := exec.CommandContext(ctx, t.Spcomp,
		"-i"+includeDir, "-i"+t.IncludeDir, "-o"+smx, sourcePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("spcomp %s: %w: %s", sourcePath, err, out)
	}
	return nil
}

func (t Toolchain) run(ctx context.Context, smx string) ([]int32, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, t.Spshell, smx)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("spshell %s: %w: %s", smx, err, stderr.String())
	}
	// spshell reports an aborted plugin on stderr and still exits zero.
	if stderr.Len() != 0 {
		return nil, fmt.Errorf("spshell %s: %s", smx, strings.TrimSpace(stderr.String()))
	}
	return parseCells(stdout.String())
}

func parseCells(out string) ([]int32, error) {
	fields := strings.Fields(out)
	cells := make([]int32, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.ParseInt(f, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("spshell printed %q, wanted a cell: %w", f, err)
		}
		// ParseInt with bitSize 32 bounds this to int32, so the cast is lossless.
		cells = append(cells, int32(v))
	}
	return cells, nil
}

func inputsInclude(inputs []Triple) string {
	var b strings.Builder
	b.WriteString("float gInputs[][3] = {\n")
	for _, in := range inputs {
		fmt.Fprintf(&b, "\t{%s, %s, %s},\n", literal(in[0]), literal(in[1]), literal(in[2]))
	}
	b.WriteString("};\n")
	return b.String()
}

/*
	literal writes v as a SourcePawn float literal

Shortest decimal that reads back as the same float32, in the form spcomp's lexer
takes. Its rules are narrower than Go's, and getting them wrong is silent: a
literal read back as a different float is a wrong answer in every generated
function that carries it, with no diagnostic anywhere.

Plain decimal for anything Go writes that way. Exponent form for the rest,
because 'f' form is where this went wrong: FLT_MAX as 39 digits compiled to
0x5f794ad1, about 1.8e19, and spcomp said nothing.

Two rules for the exponent form. The mantissa needs a point with a digit each
side, so 1e-45 is "number literal has invalid digits" and 1.0e-45 is the
smallest denormal. And a positive exponent carries no sign, so 3.4028235e+38 is
"exponential must be followed by integer".
*/
func literal(v float32) string {
	s := strconv.FormatFloat(float64(v), 'g', -1, 32)
	mantissa, exponent, hasExponent := strings.Cut(s, "e")
	if !strings.ContainsRune(mantissa, '.') {
		mantissa += ".0"
	}
	if !hasExponent {
		return mantissa
	}
	return mantissa + "e" + strings.TrimPrefix(exponent, "+")
}

/*
	WithSourceMod swaps in the compiler the plugin actually ships with

The differential tests compile with the 1.13 spcomp built beside spshell, and
the plugin builds with SourceMod's 1.12 spcomp64. Same lineage, but a check that
does not cover what ships is not a check, so mvm-z83.13 asks for both.

Only the compiler changes. The VM stays spshell's, which works because the
generated table is integer arithmetic: SourceMod's spcomp implicitly includes
its own float.inc naming the float operators __FLOAT_DIV where spshell binds
them __float_div, so anything that divides dies under the other one's VM.
*/
func (t Toolchain) WithSourceMod(upstream string) (Toolchain, error) {
	sm := filepath.Join(upstream, "testbed", "build", "spcomp", "addons", "sourcemod", "scripting")
	t.Spcomp = filepath.Join(sm, "spcomp64")
	t.IncludeDir = filepath.Join(sm, "include")
	for _, path := range []string{t.Spcomp, t.IncludeDir} {
		if _, err := os.Stat(path); err != nil {
			return Toolchain{}, fmt.Errorf("%w: %s: run testbed/build.sh", ErrNoToolchain, path)
		}
	}
	return t, nil
}
