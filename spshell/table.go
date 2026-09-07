package spshell

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// GoldenTable renders rows of a Go struct type as a SourcePawn golden-input
// table: one flat array of cells, one row per case, plus a named constant per
// column so the driver indexes by field name rather than by a number nobody
// can check.
//
// A flat array rather than an array of enum structs, because SourcePawn has no
// initialiser for an array of enum structs and the driver would have to be a
// filling function the size of the table.
//
// bool comes out as 0 or 1, an int32 as itself, and a float32 as its bits, so
// the driver reads it back with view_as<float> and nothing is rounded on the
// way in. Any other field type is refused: there is no cell for it.
func GoldenTable[T comparable](varName string, rows []T) (string, error) {
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Struct {
		return "", fmt.Errorf("spshell: %s is not a struct", typ)
	}
	if typ.NumField() == 0 {
		return "", fmt.Errorf("spshell: %s has no fields", typ)
	}
	for i := range typ.NumField() {
		if err := checkFieldType(typ.Field(i)); err != nil {
			return "", err
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "/* Generated golden inputs: %d rows of %s. */\n", len(rows), typ)
	fmt.Fprintf(&b, "#define %s_COLUMNS %d\n", strings.ToUpper(varName), typ.NumField())
	fmt.Fprintf(&b, "#define %s_ROWS %d\n\n", strings.ToUpper(varName), len(rows))
	b.WriteString("enum\n{\n")
	for i := range typ.NumField() {
		comma := ","
		if i == typ.NumField()-1 {
			comma = ""
		}
		fmt.Fprintf(&b, "\t%s_%s = %d%s\n", strings.ToUpper(varName), typ.Field(i).Name, i, comma)
	}
	b.WriteString("};\n\n")

	fmt.Fprintf(&b, "int %s[%s_ROWS][%s_COLUMNS] = {\n", varName, strings.ToUpper(varName), strings.ToUpper(varName))
	cells := make([]string, typ.NumField())
	for _, row := range rows {
		v := reflect.ValueOf(row)
		for i := range typ.NumField() {
			cells[i] = cell(v.Field(i))
		}
		b.WriteString("\t{" + strings.Join(cells, ", ") + "},\n")
	}
	b.WriteString("};\n")
	return b.String(), nil
}

func checkFieldType(f reflect.StructField) error {
	switch f.Type.Kind() {
	case reflect.Bool, reflect.Int32, reflect.Float32:
		return nil
	}
	return fmt.Errorf("spshell: the field %s is a %s, and a SourcePawn cell holds bool, int32 or float32", f.Name, f.Type)
}

func cell(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return "1"
		}
		return "0"
	case reflect.Float32:
		bits := math.Float32bits(float32(v.Float()))
		return strconv.FormatInt(int64(int32(bits)), 10) //nolint:gosec // G115: reinterpreting 32 bits as 32 bits
	default:
		return strconv.FormatInt(v.Int(), 10)
	}
}
