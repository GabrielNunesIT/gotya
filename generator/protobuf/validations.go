package protobuf

import (
	"fmt"
	"strings"

	"github.com/gotya/gotya/schema"
)

// buildValidateOptions constructs standard or CEL-based bufbuild/protovalidate
// options based on YANG schema constraints.
// It returns the inner string to be placed inside field option brackets,
// e.g., `(buf.validate.field).string = { min_len: 1 }`.
func buildValidateOptions(td schema.TypeDefinition, repeated bool, protoBaseType string) string {
	var rules []string

	// Handle Pattern constraints (assuming string)
	if len(td.Pattern) > 0 && protoBaseType == "string" {
		if len(td.Pattern) == 1 {
			pattern := td.Pattern[0]
			if !strings.HasPrefix(pattern, "^") {
				pattern = "^" + pattern
			}
			if !strings.HasSuffix(pattern, "$") {
				pattern = pattern + "$"
			}
			rules = append(rules, fmt.Sprintf("pattern: %q", pattern))
		} else {
			var parts []string
			for _, pat := range td.Pattern {
				if !strings.HasPrefix(pat, "^") {
					pat = "^" + pat
				}
				if !strings.HasSuffix(pat, "$") {
					pat = pat + "$"
				}
				parts = append(parts, fmt.Sprintf("this.matches(%q)", pat))
			}
			celRule := fmt.Sprintf("{ id: %q, expression: %q }", "pattern", strings.Join(parts, " && "))
			return formatFieldRule(repeated, "cel", celRule)
		}
	}

	// Handle Length constraints
	if len(td.Length) > 0 && (protoBaseType == "string" || protoBaseType == "bytes") {
		lengthExpr := td.Length[0]
		min, max, isMulti := parseBounds(lengthExpr)
		if isMulti {
			celExpr := translateBoundsToCEL(lengthExpr, "this.size()")
			if celExpr != "" {
				celRule := fmt.Sprintf("{ id: %q, expression: %q }", "length", celExpr)
				return formatFieldRule(repeated, "cel", celRule)
			}
		} else if min != "" || max != "" {
			if min == max {
				rules = append(rules, fmt.Sprintf("len: %s", min))
			} else {
				if min != "" && min != "min" {
					if protoBaseType == "bytes" && min == "0" {
					} else {
						rules = append(rules, fmt.Sprintf("min_len: %s", min))
					}
				}
				if max != "" && max != "max" {
					rules = append(rules, fmt.Sprintf("max_len: %s", max))
				}
			}
		}
	}

	// Handle Range constraints
	if len(td.Range) > 0 && isNumericProtoBase(protoBaseType) {
		rangeExpr := td.Range[0]
		min, max, isMulti := parseBounds(rangeExpr)
		if isMulti {
			celExpr := translateBoundsToCEL(rangeExpr, "this")
			if celExpr != "" {
				celRule := fmt.Sprintf("{ id: %q, expression: %q }", "range", celExpr)
				return formatFieldRule(repeated, "cel", celRule)
			}
		} else if min != "" || max != "" {
			if min == max {
				rules = append(rules, fmt.Sprintf("in: [%s]", min))
			} else {
				if min != "" && min != "min" {
					rules = append(rules, fmt.Sprintf("gte: %s", min))
				}
				if max != "" && max != "max" {
					rules = append(rules, fmt.Sprintf("lte: %s", max))
				}
			}
		}
	}

	if len(rules) > 0 {
		return formatFieldRule(repeated, protoBaseType, "{ "+strings.Join(rules, ", ")+" }")
	}

	return ""
}

func isNumericProtoBase(t string) bool {
	switch t {
	case "int32", "int64", "uint32", "uint64", "float", "double", "fixed32", "fixed64", "sfixed32", "sfixed64", "sint32", "sint64":
		return true
	}
	return false
}

func formatFieldRule(repeated bool, typ, rule string) string {
	if repeated {
		return fmt.Sprintf("(buf.validate.field).repeated.items.%s = %s", typ, rule)
	}
	return fmt.Sprintf("(buf.validate.field).%s = %s", typ, rule)
}

// parseBounds parses a YANG length/range expression and returns the min, max, and whether it has multiple parts using '|'.
func parseBounds(expr string) (string, string, bool) {
	expr = strings.ReplaceAll(expr, " ", "")
	if strings.Contains(expr, "|") {
		return "", "", true
	}
	parts := strings.Split(expr, "..")
	if len(parts) == 1 {
		return parts[0], parts[0], false
	}
	return parts[0], parts[1], false
}

// translateBoundsToCEL converts a complex YANG constraint with '|' to a CEL expression.
func translateBoundsToCEL(expr string, varName string) string {
	expr = strings.ReplaceAll(expr, " ", "")
	parts := strings.Split(expr, "|")
	var conditions []string

	for _, part := range parts {
		rng := strings.Split(part, "..")
		if len(rng) == 1 {
			conditions = append(conditions, fmt.Sprintf("(%s == %s)", varName, rng[0]))
		} else {
			min, max := rng[0], rng[1]
			var subCond []string
			if min != "min" {
				subCond = append(subCond, fmt.Sprintf("%s >= %s", varName, min))
			}
			if max != "max" {
				subCond = append(subCond, fmt.Sprintf("%s <= %s", varName, max))
			}
			if len(subCond) > 0 {
				conditions = append(conditions, "("+strings.Join(subCond, " && ")+")")
			}
		}
	}

	if len(conditions) > 0 {
		return strings.Join(conditions, " || ")
	}
	return ""
}
