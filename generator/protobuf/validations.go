package protobuf

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GabrielNunesIT/gotya/schema"
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
			parts := make([]string, 0, len(td.Pattern))
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
		minVal, maxVal, isMulti := parseBounds(lengthExpr)
		if isMulti {
			celExpr := translateBoundsToCEL(lengthExpr, "this.size()")
			if celExpr != "" {
				celRule := fmt.Sprintf("{ id: %q, expression: %q }", "length", celExpr)
				return formatFieldRule(repeated, "cel", celRule)
			}
		} else if minVal != "" || maxVal != "" {
			if minVal == maxVal {
				rules = append(rules, "len: "+minVal)
			} else {
				if minVal != "" && minVal != "min" {
					if protoBaseType != "bytes" || minVal != "0" {
						rules = append(rules, "min_len: "+minVal)
					}
				}
				if maxVal != "" && maxVal != "max" {
					rules = append(rules, "max_len: "+maxVal)
				}
			}
		}
	}

	// Handle Range constraints
	if len(td.Range) > 0 && isNumericProtoBase(protoBaseType) {
		rangeExpr := td.Range[0]
		minVal, maxVal, isMulti := parseBounds(rangeExpr)
		if isMulti {
			celExpr := translateBoundsToCEL(rangeExpr, "this")
			if celExpr != "" {
				celRule := fmt.Sprintf("{ id: %q, expression: %q }", "range", celExpr)
				return formatFieldRule(repeated, "cel", celRule)
			}
		} else if minVal != "" || maxVal != "" {
			if minVal == maxVal {
				rules = append(rules, fmt.Sprintf("in: [%s]", minVal))
			} else {
				if minVal != "" && minVal != "min" {
					rules = append(rules, "gte: "+minVal)
				}
				if maxVal != "" && maxVal != "max" {
					rules = append(rules, "lte: "+maxVal)
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

// collectCELPathErrors validates that each annotated field (by proto field name) exists somewhere
// in the corresponding module's schema tree. Returns all invalid paths — nil if all valid.
// annotatedFields maps protoFieldName -> moduleName for fields that received CEL annotations.
func collectCELPathErrors(modules []*schema.Module, annotatedFields map[string]string) []string {
	var errs []string
	for protoFieldName, modName := range annotatedFields {
		var found bool
		for _, mod := range modules {
			if mod.Name == modName && findNodeByYangName(mod.Nodes, protoFieldName) != nil {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Sprintf("module %s: CEL annotation path %q has no corresponding schema node", modName, protoFieldName))
		}
	}
	sort.Strings(errs)
	return errs
}

// findNodeByYangName recursively searches nodes for a node whose name matches the given name.
func findNodeByYangName(nodes map[string]schema.Node, name string) schema.Node {
	if n, ok := nodes[name]; ok {
		return n
	}
	for _, node := range nodes {
		if children := node.GetChildren(); len(children) > 0 {
			if found := findNodeByYangName(children, name); found != nil {
				return found
			}
		}
	}
	return nil
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
			minVal, maxVal := rng[0], rng[1]
			var subCond []string
			if minVal != "min" {
				subCond = append(subCond, fmt.Sprintf("%s >= %s", varName, minVal))
			}
			if maxVal != "max" {
				subCond = append(subCond, fmt.Sprintf("%s <= %s", varName, maxVal))
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
