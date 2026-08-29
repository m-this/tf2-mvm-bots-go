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

	for name, body := range includes {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return nil, fmt.Errorf("writing %s: %w", path, err)
		}
	}

	smx := filepath.Join(dir, "golden.smx")
	if err := t.compile(ctx, sourcePath, smx, dir); err != nil {
		return nil, err
	}
	return t.run(ctx, smx)
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

// literal writes v as a SourcePawn float literal: shortest decimal that reads
// back as the same float32, always with a point, never in exponent form, which
// the compiler's lexer does not take for floats.
func literal(v float32) string {
	s := strconv.FormatFloat(float64(v), 'f', -1, 32)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return s
}
