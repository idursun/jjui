package scripting

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type tomlStructField struct {
	name  string
	index int
}

func toLuaTable[T any](L *lua.LState, value T) *lua.LTable {
	v := structToLuaValue(L, reflect.ValueOf(value))
	tbl, ok := v.(*lua.LTable)
	if !ok {
		return L.NewTable()
	}
	return tbl
}

func fromLuaTable[T any](tbl *lua.LTable, out *T) error {
	if out == nil {
		return fmt.Errorf("config: destination is nil")
	}
	if tbl == nil {
		return fmt.Errorf("config: expected table, got nil")
	}
	return assignLuaToValue(reflect.ValueOf(out).Elem(), tbl, "config")
}

func structToLuaValue(L *lua.LState, v reflect.Value) lua.LValue {
	if !v.IsValid() {
		return lua.LNil
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return lua.LNil
		}
		return structToLuaValue(L, v.Elem())

	case reflect.Interface:
		if v.IsNil() {
			return lua.LNil
		}
		return structToLuaValue(L, v.Elem())

	case reflect.Struct:
		tbl := L.NewTable()
		for _, field := range tomlStructFields(v.Type()) {
			tbl.RawSetString(field.name, structToLuaValue(L, v.Field(field.index)))
		}
		return tbl

	case reflect.Slice, reflect.Array:
		tbl := L.NewTable()
		for i := 0; i < v.Len(); i++ {
			tbl.Append(structToLuaValue(L, v.Index(i)))
		}
		return tbl

	case reflect.Map:
		tbl := L.NewTable()
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := structToLuaValue(L, iter.Value())
			if key.Kind() == reflect.String {
				tbl.RawSetString(key.String(), val)
				continue
			}
			tbl.RawSet(lua.LString(fmt.Sprint(key.Interface())), val)
		}
		return tbl

	case reflect.String:
		return lua.LString(v.String())
	case reflect.Bool:
		return lua.LBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(v.Int())
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(v.Float())
	}

	return lua.LNil
}

func assignLuaToValue(dst reflect.Value, value lua.LValue, path string) error {
	if !dst.IsValid() || value == lua.LNil {
		return nil
	}

	switch dst.Kind() {
	case reflect.Pointer:
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return assignLuaToValue(dst.Elem(), value, path)

	case reflect.Struct:
		tbl, ok := value.(*lua.LTable)
		if !ok {
			return luaTypeError(path, "table", value)
		}
		return assignLuaTableToStruct(dst, tbl, path)

	case reflect.Map:
		tbl, ok := value.(*lua.LTable)
		if !ok {
			return luaTypeError(path, "table", value)
		}
		if dst.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("%s: unsupported map key type %s", path, dst.Type().Key())
		}
		if dst.IsNil() {
			dst.Set(reflect.MakeMap(dst.Type()))
		}
		keyType := dst.Type().Key()
		elemType := dst.Type().Elem()
		var assignErr error
		tbl.ForEach(func(key, val lua.LValue) {
			if assignErr != nil {
				return
			}

			keyStr, ok := key.(lua.LString)
			if !ok {
				return
			}
			mapKey := reflect.ValueOf(keyStr.String()).Convert(keyType)

			elem := reflect.New(elemType).Elem()
			if existing := dst.MapIndex(mapKey); existing.IsValid() {
				elem.Set(existing)
			}

			entryPath := path + "." + keyStr.String()
			if err := assignLuaToValue(elem, val, entryPath); err != nil {
				assignErr = err
				return
			}
			dst.SetMapIndex(mapKey, elem)
		})
		return assignErr

	case reflect.Slice:
		tbl, ok := value.(*lua.LTable)
		if !ok {
			return luaTypeError(path, "table", value)
		}
		length := tbl.Len()
		out := reflect.MakeSlice(dst.Type(), length, length)
		for i := 1; i <= length; i++ {
			if err := assignLuaToValue(out.Index(i-1), tbl.RawGetInt(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		dst.Set(out)
		return nil

	case reflect.Interface:
		converted := luaValueToGo(value)
		if converted == nil {
			dst.SetZero()
			return nil
		}

		rv := reflect.ValueOf(converted)
		if rv.Type().AssignableTo(dst.Type()) {
			dst.Set(rv)
			return nil
		}
		if dst.Type().NumMethod() == 0 {
			dst.Set(rv)
			return nil
		}
		if rv.Type().Implements(dst.Type()) {
			dst.Set(rv)
			return nil
		}
		return fmt.Errorf("%s: expected %s, got %s", path, dst.Type(), value.Type().String())

	case reflect.Array:
		return fmt.Errorf("%s: arrays are not supported", path)

	case reflect.String:
		s, ok := value.(lua.LString)
		if !ok {
			return luaTypeError(path, "string", value)
		}
		dst.SetString(s.String())
		return nil

	case reflect.Bool:
		b, ok := value.(lua.LBool)
		if !ok {
			return luaTypeError(path, "boolean", value)
		}
		dst.SetBool(bool(b))
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, ok := value.(lua.LNumber)
		if !ok {
			return luaTypeError(path, "integer", value)
		}
		f := float64(n)
		if math.Trunc(f) != f {
			return luaTypeError(path, "integer", value)
		}
		iv := int64(f)
		if dst.OverflowInt(iv) {
			return fmt.Errorf("%s: integer %d overflows %s", path, iv, dst.Type())
		}
		dst.SetInt(iv)
		return nil

	case reflect.Float32, reflect.Float64:
		n, ok := value.(lua.LNumber)
		if !ok {
			return luaTypeError(path, "number", value)
		}
		dst.SetFloat(float64(n))
		return nil
	}

	return nil
}

func assignLuaTableToStruct(dst reflect.Value, tbl *lua.LTable, path string) error {
	fields := make(map[string]int, dst.NumField())
	for _, field := range tomlStructFields(dst.Type()) {
		fields[field.name] = field.index
	}

	var assignErr error
	tbl.ForEach(func(key, value lua.LValue) {
		if assignErr != nil {
			return
		}
		keyStr, ok := key.(lua.LString)
		if !ok {
			return
		}
		idx, ok := fields[keyStr.String()]
		if !ok {
			return
		}
		assignErr = assignLuaToValue(dst.Field(idx), value, path+"."+keyStr.String())
	})
	return assignErr
}

func luaTypeError(path, expected string, got lua.LValue) error {
	return fmt.Errorf("%s: expected %s, got %s", path, expected, got.Type().String())
}

func tomlFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("toml")
	if tag == "" || tag == "-" {
		return "", false
	}

	if idx := strings.IndexByte(tag, ','); idx >= 0 {
		tag = tag[:idx]
	}
	if tag == "" || tag == "-" {
		return "", false
	}

	return tag, true
}

func tomlStructFields(typ reflect.Type) []tomlStructField {
	fields := make([]tomlStructField, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, ok := tomlFieldName(field)
		if !ok {
			continue
		}
		fields = append(fields, tomlStructField{name: name, index: i})
	}
	return fields
}
