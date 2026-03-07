//nolint:testpackage
package protobuf

import (
	"reflect"
	"testing"

	"github.com/gotya/gotya/schema"
)

func TestBuildValidateOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		td            schema.TypeDefinition
		repeated      bool
		protoBaseType string
		want          string
	}{
		{
			name: "single length string",
			td: schema.TypeDefinition{
				Length: []string{"1..255"},
			},
			protoBaseType: "string",
			want:          `(buf.validate.field).string = { min_len: 1, max_len: 255 }`,
		},
		{
			name: "exact length bytes",
			td: schema.TypeDefinition{
				Length: []string{"128"},
			},
			protoBaseType: "bytes",
			want:          `(buf.validate.field).bytes = { len: 128 }`,
		},
		{
			name: "multi length cel",
			td: schema.TypeDefinition{
				Length: []string{"10 | 20..30"},
			},
			protoBaseType: "string",
			want:          `(buf.validate.field).cel = { id: "length", expression: "(this.size() == 10) || (this.size() >= 20 && this.size() <= 30)" }`,
		},
		{
			name: "single range int32",
			td: schema.TypeDefinition{
				Range: []string{"1..100"},
			},
			protoBaseType: "int32",
			want:          `(buf.validate.field).int32 = { gte: 1, lte: 100 }`,
		},
		{
			name: "single range without upper bound int32",
			td: schema.TypeDefinition{
				Range: []string{"1..max"},
			},
			protoBaseType: "int32",
			want:          `(buf.validate.field).int32 = { gte: 1 }`,
		},
		{
			name: "single range without lower bound int32",
			td: schema.TypeDefinition{
				Range: []string{"min..100"},
			},
			protoBaseType: "int32",
			want:          `(buf.validate.field).int32 = { lte: 100 }`,
		},
		{
			name: "multi range cel",
			td: schema.TypeDefinition{
				Range: []string{"1 | 10..20"},
			},
			protoBaseType: "int32",
			want:          `(buf.validate.field).cel = { id: "range", expression: "(this == 1) || (this >= 10 && this <= 20)" }`,
		},
		{
			name: "single pattern string",
			td: schema.TypeDefinition{
				Pattern: []string{"[a-z]+"},
			},
			protoBaseType: "string",
			want:          `(buf.validate.field).string = { pattern: "^[a-z]+$" }`,
		},
		{
			name: "single pattern already anchored string",
			td: schema.TypeDefinition{
				Pattern: []string{"^[a-z]+$"},
			},
			protoBaseType: "string",
			want:          `(buf.validate.field).string = { pattern: "^[a-z]+$" }`,
		},
		{
			name: "multi pattern cel",
			td: schema.TypeDefinition{
				Pattern: []string{"[A-Z]+", "[a-z]+"},
			},
			protoBaseType: "string",
			want:          `(buf.validate.field).cel = { id: "pattern", expression: "this.matches(\"^[A-Z]+$\") && this.matches(\"^[a-z]+$\")" }`,
		},
		{
			name: "repeated string range",
			td: schema.TypeDefinition{
				Length: []string{"1..100"},
			},
			repeated:      true,
			protoBaseType: "string",
			want:          `(buf.validate.field).repeated.items.string = { min_len: 1, max_len: 100 }`,
		},
		{
			name: "repeated numeric cel range",
			td: schema.TypeDefinition{
				Range: []string{"1 | 100"},
			},
			repeated:      true,
			protoBaseType: "uint32",
			want:          `(buf.validate.field).repeated.items.cel = { id: "range", expression: "(this == 1) || (this == 100)" }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := buildValidateOptions(tt.td, tt.repeated, tt.protoBaseType); got != tt.want {
				t.Errorf("buildValidateOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expr    string
		min     string
		max     string
		isMulti bool
	}{
		{"1..100", "1", "100", false},
		{" 1 .. 100 ", "1", "100", false},
		{"10", "10", "10", false},
		{"1..10 | 20..30", "", "", true},
		{"min..10", "min", "10", false},
		{"10..max", "10", "max", false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			gotMin, gotMax, gotIsMulti := parseBounds(tt.expr)
			if gotMin != tt.min {
				t.Errorf("parseBounds() gotMin = %v, want %v", gotMin, tt.min)
			}
			if gotMax != tt.max {
				t.Errorf("parseBounds() gotMax = %v, want %v", gotMax, tt.max)
			}
			if gotIsMulti != tt.isMulti {
				t.Errorf("parseBounds() gotIsMulti = %v, want %v", gotIsMulti, tt.isMulti)
			}
		})
	}
}

func TestTranslateBoundsToCEL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expr string
		want string
	}{
		{"1..10 | 20..30", "(this >= 1 && this <= 10) || (this >= 20 && this <= 30)"},
		{"1 | 10..20", "(this == 1) || (this >= 10 && this <= 20)"},
		{"min..10 | 20..max", "(this <= 10) || (this >= 20)"},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			if got := translateBoundsToCEL(tt.expr, "this"); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("translateBoundsToCEL() = %v, want %v", got, tt.want)
			}
		})
	}
}
