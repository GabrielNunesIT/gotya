// Package rfc7951 implements JSON encoding and decoding for YANG data
// as specified in RFC 7951 (JSON Encoding of Data Modeled with YANG).
package rfc7951

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ErrSkipValue is a sentinel error to indicate that a field, slice, or map should be omitted.
var ErrSkipValue = errors.New("skip value")

// Encode marshals a gotya-generated Go struct into RFC 7951 compliant JSON.
//
// RFC 7951 rules applied:
//   - JSON member names are prefixed with the YANG module name when the
//     child node's module differs from its parent's module.
//   - int64 and uint64 values are encoded as JSON strings.
//   - empty types (struct{}) are encoded as [null].
//   - bool, numeric, and string values follow standard JSON encoding.
func Encode(v interface{}) ([]byte, error) {
	result, err := encodeValue(reflect.ValueOf(v), "")
	if err != nil && !errors.Is(err, ErrSkipValue) {
		return nil, fmt.Errorf("rfc7951 encode: %w", err)
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}
	return bytes, nil
}

// encodeValue recursively encodes a reflect.Value into an RFC 7951 compatible
// intermediate representation (maps, slices, strings, numbers).
// parentModule tracks the YANG module of the parent for namespace prefix decisions.
func encodeValue(v reflect.Value, parentModule string) (interface{}, error) {
	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, ErrSkipValue
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		return encodeStruct(v, parentModule)
	case reflect.Slice:
		return encodeSlice(v, parentModule)
	case reflect.Map:
		return encodeMap(v, parentModule)
	case reflect.Int64:
		// RFC 7951 §6.1: int64 encoded as string
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint64:
		// RFC 7951 §6.1: uint64 encoded as string
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return v.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return v.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return v.Float(), nil
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.String:
		return v.String(), nil
	default:
		return nil, fmt.Errorf("unsupported kind: %s", v.Kind())
	}
}

// encodeStruct encodes a Go struct into an RFC 7951 JSON object.
func encodeStruct(v reflect.Value, parentModule string) (interface{}, error) {
	// Check for empty type (struct{}) — RFC 7951 §6.9: encoded as [null]
	if v.Type().NumField() == 0 {
		return []interface{}{nil}, nil
	}

	t := v.Type()
	result := make(map[string]interface{})

	for i := range t.NumField() {
		field := t.Field(i)
		fieldVal := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip nil pointers and empty slices/maps (omitempty behavior)
		if isZeroValue(fieldVal) {
			continue
		}

		// Parse the yang tag to get module:name
		yangTag := field.Tag.Get("yang")
		if yangTag == "" {
			// Fallback to json tag for fields without yang tags (e.g., YANGAnnotations)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			name := strings.Split(jsonTag, ",")[0]
			encoded, err := encodeValue(fieldVal, parentModule)
			if err != nil {
				if errors.Is(err, ErrSkipValue) {
					continue
				}
				return nil, fmt.Errorf("field %s: %w", field.Name, err)
			}
			if encoded != nil {
				result[name] = encoded
			}
			continue
		}

		// Parse yang:"module:name"
		parts := strings.SplitN(yangTag, ":", 2)
		if len(parts) != 2 {
			continue
		}
		fieldModule := parts[0]
		fieldName := parts[1]

		// RFC 7951 §4: prefix with module name when crossing namespace boundaries
		key := fieldName
		if fieldModule != parentModule {
			key = fieldModule + ":" + fieldName
		}

		encoded, err := encodeValue(fieldVal, fieldModule)
		if err != nil {
			if errors.Is(err, ErrSkipValue) {
				continue
			}
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}
		if encoded != nil {
			result[key] = encoded
		}
	}

	if len(result) == 0 {
		return nil, ErrSkipValue
	}
	return result, nil
}

// encodeSlice encodes a Go slice into a JSON array.
func encodeSlice(v reflect.Value, parentModule string) (interface{}, error) {
	if v.IsNil() || v.Len() == 0 {
		return nil, ErrSkipValue
	}

	result := make([]interface{}, 0, v.Len())
	for i := range v.Len() {
		encoded, err := encodeValue(v.Index(i), parentModule)
		if err != nil {
			if errors.Is(err, ErrSkipValue) {
				continue
			}
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		result = append(result, encoded)
	}
	return result, nil
}

// encodeMap encodes a Go map into a JSON array of objects (for YANG lists keyed by string).
func encodeMap(v reflect.Value, parentModule string) (interface{}, error) {
	if v.IsNil() || v.Len() == 0 {
		return nil, ErrSkipValue
	}

	result := make([]interface{}, 0, v.Len())
	for _, key := range v.MapKeys() {
		encoded, err := encodeValue(v.MapIndex(key), parentModule)
		if err != nil {
			if errors.Is(err, ErrSkipValue) {
				continue
			}
			return nil, fmt.Errorf("map key %v: %w", key, err)
		}
		if encoded != nil {
			result = append(result, encoded)
		}
	}
	return result, nil
}

// isZeroValue returns true for nil pointers, empty slices, empty maps, and zero-value scalars.
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	default:
		return false
	}
}
