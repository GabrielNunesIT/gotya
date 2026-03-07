package rfc7951_test

import (
	"encoding/json"
	"testing"

	"github.com/gotya/gotya/codec/rfc7951"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test structs simulating gotya-generated output ---

// EmptyType simulates an empty YANG leaf.
type EmptyType struct{}

// SimpleLeaf simulates a container with basic leaf types.
type SimpleLeaf struct {
	Name    *string    `json:"name,omitempty" yang:"test-module:name"`
	Counter *int64     `json:"counter,omitempty" yang:"test-module:counter"`
	BigVal  *uint64    `json:"big-val,omitempty" yang:"test-module:big-val"`
	Active  *bool      `json:"active,omitempty" yang:"test-module:active"`
	Rate    *float64   `json:"rate,omitempty" yang:"test-module:rate"`
	Marker  *EmptyType `json:"marker,omitempty" yang:"test-module:marker"`
}

// ChildContainer simulates a child from a different module (cross-namespace).
type ChildContainer struct {
	Speed *int32 `json:"speed,omitempty" yang:"other-module:speed"`
}

// ParentContainer simulates a parent with a cross-module child.
type ParentContainer struct {
	Description *string         `json:"description,omitempty" yang:"test-module:description"`
	Child       *ChildContainer `json:"child,omitempty" yang:"other-module:child"`
	Local       *ChildContainer `json:"local,omitempty" yang:"test-module:local"`
}

// ListParent simulates a YANG list parent using ordered slices.
type ListEntry struct {
	Key   *string `json:"key,omitempty" yang:"test-module:key"`
	Value *string `json:"value,omitempty" yang:"test-module:value"`
}

type ListParent struct {
	Entries []*ListEntry `json:"entries,omitempty" yang:"test-module:entries"`
}

func ptrStr(s string) *string       { return &s }
func ptrInt64(n int64) *int64       { return &n }
func ptrUint64(n uint64) *uint64    { return &n }
func ptrBool(b bool) *bool          { return &b }
func ptrFloat64(f float64) *float64 { return &f }
func ptrInt32(n int32) *int32       { return &n }

func TestEncode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name: "int64 encoded as string",
			input: SimpleLeaf{
				Counter: ptrInt64(9223372036854775807),
			},
			expected: `{"test-module:counter":"9223372036854775807"}`,
		},
		{
			name: "uint64 encoded as string",
			input: SimpleLeaf{
				BigVal: ptrUint64(18446744073709551615),
			},
			expected: `{"test-module:big-val":"18446744073709551615"}`,
		},
		{
			name: "empty type encoded as [null]",
			input: SimpleLeaf{
				Marker: &EmptyType{},
			},
			expected: `{"test-module:marker":[null]}`,
		},
		{
			name: "basic string and bool",
			input: SimpleLeaf{
				Name:   ptrStr("eth0"),
				Active: ptrBool(true),
			},
			expected: `{"test-module:active":true,"test-module:name":"eth0"}`,
		},
		{
			name: "cross-namespace prefixing",
			input: ParentContainer{
				Description: ptrStr("parent desc"),
				Child:       &ChildContainer{Speed: ptrInt32(1000)},
				Local:       &ChildContainer{Speed: ptrInt32(500)},
			},
			// At root level, parent has no module so all fields get prefixed.
			// Inside ParentContainer (test-module), "child" from other-module gets prefixed,
			// "local" from test-module also gets prefixed since parent module is "".
			// Inside ChildContainer (other-module), "speed" belongs to other-module.
			expected: `{"test-module:description":"parent desc","test-module:local":{"other-module:speed":500},"other-module:child":{"speed":1000}}`,
		},
		{
			name: "list encoded as array",
			input: ListParent{
				Entries: []*ListEntry{
					{Key: ptrStr("a"), Value: ptrStr("1")},
					{Key: ptrStr("b"), Value: ptrStr("2")},
				},
			},
			expected: `{"test-module:entries":[{"key":"a","value":"1"},{"key":"b","value":"2"}]}`,
		},
		{
			name:     "nil fields omitted",
			input:    SimpleLeaf{},
			expected: `null`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := rfc7951.Encode(tc.input)
			require.NoError(t, err)

			// Normalize JSON by re-marshaling expected
			var expectedObj interface{}
			err = json.Unmarshal([]byte(tc.expected), &expectedObj)
			require.NoError(t, err, "invalid expected JSON")

			var gotObj interface{}
			err = json.Unmarshal(got, &gotObj)
			require.NoError(t, err, "invalid encoded JSON")

			assert.Equal(t, expectedObj, gotObj)
		})
	}
}

func TestDecode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		target   interface{}
		expected interface{}
	}{
		{
			name:   "int64 decoded from string",
			input:  `{"counter":"9223372036854775807"}`,
			target: &SimpleLeaf{},
			expected: &SimpleLeaf{
				Counter: ptrInt64(9223372036854775807),
			},
		},
		{
			name:   "uint64 decoded from string",
			input:  `{"big-val":"18446744073709551615"}`,
			target: &SimpleLeaf{},
			expected: &SimpleLeaf{
				BigVal: ptrUint64(18446744073709551615),
			},
		},
		{
			name:   "basic string and bool",
			input:  `{"name":"eth0","active":true}`,
			target: &SimpleLeaf{},
			expected: &SimpleLeaf{
				Name:   ptrStr("eth0"),
				Active: ptrBool(true),
			},
		},
		{
			name:   "cross-namespace key with prefix",
			input:  `{"description":"parent desc","other-module:child":{"other-module:speed":1000}}`,
			target: &ParentContainer{},
			expected: &ParentContainer{
				Description: ptrStr("parent desc"),
				Child:       &ChildContainer{Speed: ptrInt32(1000)},
			},
		},
		{
			name:   "list decoded from array",
			input:  `{"entries":[{"key":"a","value":"1"},{"key":"b","value":"2"}]}`,
			target: &ListParent{},
			expected: &ListParent{
				Entries: []*ListEntry{
					{Key: ptrStr("a"), Value: ptrStr("1")},
					{Key: ptrStr("b"), Value: ptrStr("2")},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := rfc7951.Decode([]byte(tc.input), tc.target)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, tc.target)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	original := SimpleLeaf{
		Name:    ptrStr("interface1"),
		Counter: ptrInt64(-42),
		BigVal:  ptrUint64(999999999999999),
		Active:  ptrBool(false),
		Rate:    ptrFloat64(3.14),
	}

	encoded, err := rfc7951.Encode(original)
	require.NoError(t, err)

	var decoded SimpleLeaf
	err = rfc7951.Decode(encoded, &decoded)
	require.NoError(t, err)

	assert.Equal(t, *original.Name, *decoded.Name)
	assert.Equal(t, *original.Counter, *decoded.Counter)
	assert.Equal(t, *original.BigVal, *decoded.BigVal)
	assert.Equal(t, *original.Active, *decoded.Active)
	assert.Equal(t, *original.Rate, *decoded.Rate)
}
