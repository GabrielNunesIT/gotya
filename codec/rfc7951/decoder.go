package rfc7951

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Decode unmarshals RFC 7951 compliant JSON into a gotya-generated Go struct.
//
// It handles:
//   - Module-prefixed JSON keys (e.g., "ietf-interfaces:interfaces")
//   - int64/uint64 values encoded as JSON strings
//   - empty types encoded as [null]
func Decode(data []byte, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("rfc7951 decode: target must be a non-nil pointer")
	}

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("rfc7951 decode: json unmarshal: %w", err)
	}

	return decodeValue(raw, rv.Elem())
}

// decodeValue recursively decodes a generic JSON value into the target reflect.Value.
func decodeValue(src interface{}, dst reflect.Value) error {
	if src == nil {
		return nil
	}

	// Allocate through pointer chains
	for dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}

	switch dst.Kind() {
	case reflect.Struct:
		return decodeStruct(src, dst)
	case reflect.Slice:
		return decodeSlice(src, dst)
	case reflect.Map:
		return decodeMap(src, dst)
	default:
		return decodeScalar(src, dst)
	}
}

// decodeStruct decodes a JSON object (map[string]interface{}) into a Go struct.
func decodeStruct(src interface{}, dst reflect.Value) error {
	// Handle empty type: [null] → struct{}
	if arr, ok := src.([]interface{}); ok {
		if len(arr) == 1 && arr[0] == nil && dst.Type().NumField() == 0 {
			return nil // empty type successfully decoded
		}
	}

	srcMap, ok := src.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected JSON object for struct %s, got %T", dst.Type().Name(), src)
	}

	t := dst.Type()

	// Build a lookup from yang node name → field index
	yangIndex := make(map[string]int)
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		yangTag := field.Tag.Get("yang")
		if yangTag != "" {
			parts := strings.SplitN(yangTag, ":", 2)
			if len(parts) == 2 {
				yangIndex[parts[1]] = i // match by bare name
				yangIndex[yangTag] = i  // match by module:name
			}
		}
		// Also index by json tag
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			name := strings.Split(jsonTag, ",")[0]
			if name != "" && name != "-" {
				yangIndex[name] = i
			}
		}
	}

	for key, val := range srcMap {
		fieldIdx, found := yangIndex[key]
		if !found {
			// Try stripping the module prefix
			if idx := strings.Index(key, ":"); idx != -1 {
				lookupKey := key[idx+1:]
				fieldIdx, found = yangIndex[lookupKey]
			}
		}
		if !found {
			continue // unknown field, skip
		}

		fieldVal := dst.Field(fieldIdx)
		if err := decodeValue(val, fieldVal); err != nil {
			return fmt.Errorf("field %s: %w", t.Field(fieldIdx).Name, err)
		}
	}

	return nil
}

// decodeSlice decodes a JSON array into a Go slice.
func decodeSlice(src interface{}, dst reflect.Value) error {
	srcArr, ok := src.([]interface{})
	if !ok {
		return fmt.Errorf("expected JSON array for slice, got %T", src)
	}

	elemType := dst.Type().Elem()
	slice := reflect.MakeSlice(dst.Type(), len(srcArr), len(srcArr))
	for i, item := range srcArr {
		elem := reflect.New(elemType).Elem()
		if err := decodeValue(item, elem); err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}
		slice.Index(i).Set(elem)
	}
	dst.Set(slice)
	return nil
}

// decodeMap decodes a JSON array of objects into a Go map[string]*Struct.
func decodeMap(src interface{}, dst reflect.Value) error {
	srcArr, ok := src.([]interface{})
	if !ok {
		return fmt.Errorf("expected JSON array for map, got %T", src)
	}

	if dst.IsNil() {
		dst.Set(reflect.MakeMap(dst.Type()))
	}

	elemType := dst.Type().Elem()
	for i, item := range srcArr {
		elem := reflect.New(elemType).Elem()
		if err := decodeValue(item, elem); err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}
		// Use the first key field as map key (convention for YANG list maps)
		// For now, use the index as a string key
		key := reflect.ValueOf(strconv.Itoa(i))
		dst.SetMapIndex(key, elem)
	}
	return nil
}

// decodeScalar decodes a JSON primitive into a Go scalar value.
func decodeScalar(src interface{}, dst reflect.Value) error {
	switch dst.Kind() {
	case reflect.String:
		switch v := src.(type) {
		case string:
			dst.SetString(v)
		case float64:
			dst.SetString(fmt.Sprintf("%g", v))
		default:
			dst.SetString(fmt.Sprintf("%v", v))
		}

	case reflect.Int64:
		// RFC 7951: int64 comes as a JSON string
		switch v := src.(type) {
		case string:
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid int64 string %q: %w", v, err)
			}
			dst.SetInt(n)
		case float64:
			dst.SetInt(int64(v))
		default:
			return fmt.Errorf("cannot decode %T into int64", src)
		}

	case reflect.Uint64:
		// RFC 7951: uint64 comes as a JSON string
		switch v := src.(type) {
		case string:
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid uint64 string %q: %w", v, err)
			}
			dst.SetUint(n)
		case float64:
			dst.SetUint(uint64(v))
		default:
			return fmt.Errorf("cannot decode %T into uint64", src)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		switch v := src.(type) {
		case float64:
			dst.SetInt(int64(v))
		case string:
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid int string %q: %w", v, err)
			}
			dst.SetInt(n)
		default:
			return fmt.Errorf("cannot decode %T into %s", src, dst.Kind())
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		switch v := src.(type) {
		case float64:
			dst.SetUint(uint64(v))
		case string:
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid uint string %q: %w", v, err)
			}
			dst.SetUint(n)
		default:
			return fmt.Errorf("cannot decode %T into %s", src, dst.Kind())
		}

	case reflect.Float32, reflect.Float64:
		switch v := src.(type) {
		case float64:
			dst.SetFloat(v)
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("invalid float string %q: %w", v, err)
			}
			dst.SetFloat(f)
		default:
			return fmt.Errorf("cannot decode %T into float", src)
		}

	case reflect.Bool:
		switch v := src.(type) {
		case bool:
			dst.SetBool(v)
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("invalid bool string %q: %w", v, err)
			}
			dst.SetBool(b)
		default:
			return fmt.Errorf("cannot decode %T into bool", src)
		}

	default:
		return fmt.Errorf("unsupported scalar kind: %s", dst.Kind())
	}
	return nil
}
