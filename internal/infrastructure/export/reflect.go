package export

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TableFromSlice builds a Table from a slice of structs (or pointers to
// structs). Columns come from exported fields in declaration order, labelled by
// their `json` tag when present (minus options) or the field name otherwise.
// Fields tagged `json:"-"` and unexported fields are skipped. Nested structs
// other than time.Time are skipped to keep reports flat.
//
// This lets any list endpoint be exported with a single call, reusing the exact
// DTO it already returns to the client.
func TableFromSlice(title string, data any) (*Table, error) {
	v := reflect.ValueOf(data)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("export: expected a slice, got %s", v.Kind())
	}

	elemType := v.Type().Elem()
	for elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	switch elemType.Kind() {
	case reflect.Struct:
		return tableFromStructSlice(title, v, elemType)
	case reflect.Map:
		return tableFromMapSlice(title, v)
	default:
		// Single-column fallback for slices of scalars.
		t := &Table{Title: title, Columns: []string{"valor"}}
		for i := 0; i < v.Len(); i++ {
			t.Rows = append(t.Rows, []string{cellString(deref(v.Index(i)))})
		}
		return t, nil
	}
}

func tableFromStructSlice(title string, v reflect.Value, elemType reflect.Type) (*Table, error) {
	type colDef struct {
		index []int
		label string
	}
	var cols []colDef
	for i := 0; i < elemType.NumField(); i++ {
		f := elemType.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		tag := f.Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		// Skip nested structs/slices we cannot flatten (time.Time is allowed).
		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if (ft.Kind() == reflect.Struct && ft != reflect.TypeOf(time.Time{})) ||
			ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct {
			continue
		}
		if name == "" {
			name = f.Name
		}
		cols = append(cols, colDef{index: f.Index, label: name})
	}

	t := &Table{Title: title}
	for _, c := range cols {
		t.Columns = append(t.Columns, c.label)
	}
	for i := 0; i < v.Len(); i++ {
		row := deref(v.Index(i))
		cells := make([]string, len(cols))
		for j, c := range cols {
			cells[j] = cellString(row.FieldByIndex(c.index))
		}
		t.Rows = append(t.Rows, cells)
	}
	return t, nil
}

func tableFromMapSlice(title string, v reflect.Value) (*Table, error) {
	// Stable column set: union of keys across rows, sorted.
	keySet := map[string]struct{}{}
	for i := 0; i < v.Len(); i++ {
		m := deref(v.Index(i))
		for _, k := range m.MapKeys() {
			keySet[fmt.Sprint(k.Interface())] = struct{}{}
		}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	t := &Table{Title: title, Columns: keys}
	for i := 0; i < v.Len(); i++ {
		m := deref(v.Index(i))
		cells := make([]string, len(keys))
		for j, k := range keys {
			val := m.MapIndex(reflect.ValueOf(k))
			if val.IsValid() {
				cells[j] = cellString(reflect.ValueOf(val.Interface()))
			}
		}
		t.Rows = append(t.Rows, cells)
	}
	return t, nil
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// cellString renders a single value as display text, dereferencing pointers and
// formatting times in pt-BR.
func cellString(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "Sim"
		}
		return "Não"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Struct:
		if tm, ok := v.Interface().(time.Time); ok {
			if tm.IsZero() {
				return ""
			}
			return tm.Format("02/01/2006")
		}
	}
	return fmt.Sprint(v.Interface())
}
