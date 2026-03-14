package compiler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gotya/gotya/schema"
)

// Validator performs semantic validation on a compiled schema module.
type Validator struct {
	compiler *Compiler
	module   *schema.Module
}

// NewValidator creates a semantic validator for a compiled module.
func NewValidator(c *Compiler, m *schema.Module) *Validator {
	return &Validator{
		compiler: c,
		module:   m,
	}
}

// Validate performs semantic validation on the compiled module.
// Errors are reported directly via c.addError with the appropriate sentinel.
func (v *Validator) Validate() error {
	v.validateConfigBoundaries(nil, v.module.Nodes)
	v.validateListKeys(v.module.Nodes)
	v.validateMandatoryAndDefault(v.module.Nodes)
	v.validateIdentities(v.module.Nodes)
	v.validateDefaultValues(v.module.Nodes)
	v.validateXPathSyntax(v.module.Nodes)
	return nil
}

func (v *Validator) validateConfigBoundaries(parent schema.Node, nodes map[string]schema.Node) {
	for _, node := range nodes {
		if parent != nil && !parent.Config() && node.Config() {
			v.compiler.addError(ErrConfigBoundary, fmt.Sprintf("node %s has config true but parent %s has config false", node.Name(), parent.Name()))
		}
		if len(node.GetChildren()) > 0 {
			v.validateConfigBoundaries(node, node.GetChildren())
		}
	}
}

func (v *Validator) validateListKeys(nodes map[string]schema.Node) {
	for _, node := range nodes {
		if list, ok := node.(*schema.List); ok {
			for _, key := range list.Keys {
				child, exists := list.GetChildren()[key]
				if !exists {
					v.compiler.addError(ErrListMissingKey, fmt.Sprintf("list %s key %s not found as a child leaf", list.Name(), key))
					continue
				}
				leaf, isLeaf := child.(*schema.Leaf)
				if !isLeaf {
					v.compiler.addError(ErrListMissingKey, fmt.Sprintf("list %s key %s must be a leaf", list.Name(), key))
					continue
				}
				if leaf.Type.Name == "empty" {
					v.compiler.addError(ErrListMissingKey, fmt.Sprintf("list %s key %s cannot be of type empty", list.Name(), key))
				}
			}
		}
		if len(node.GetChildren()) > 0 {
			v.validateListKeys(node.GetChildren())
		}
	}
}

func (v *Validator) validateMandatoryAndDefault(nodes map[string]schema.Node) {
	for _, node := range nodes {
		if leaf, ok := node.(*schema.Leaf); ok {
			if leaf.Mandatory && leaf.Default != nil {
				v.compiler.addError(ErrMandatoryDefault, fmt.Sprintf("leaf %s cannot be both mandatory and have a default value", leaf.Name()))
			}
		} else if choice, ok := node.(*schema.Choice); ok {
			if choice.Mandatory && choice.Default != nil {
				v.compiler.addError(ErrMandatoryDefault, fmt.Sprintf("choice %s cannot be both mandatory and have a default case", choice.Name()))
			}
		}
		if len(node.GetChildren()) > 0 {
			v.validateMandatoryAndDefault(node.GetChildren())
		}
	}
}

func (v *Validator) validateIdentities(nodes map[string]schema.Node) {
	for _, node := range nodes {
		v.checkNodeIdentities(node)
		if len(node.GetChildren()) > 0 {
			v.validateIdentities(node.GetChildren())
		}
	}
}

func (v *Validator) checkNodeIdentities(node schema.Node) {
	var typeDef *schema.TypeDefinition
	if leaf, ok := node.(*schema.Leaf); ok {
		typeDef = &leaf.Type
	} else if leafList, ok := node.(*schema.LeafList); ok {
		typeDef = &leafList.Type
	}

	if typeDef != nil && typeDef.Name == "identityref" {
		if len(typeDef.Bases) == 0 {
			v.compiler.addError(ErrIdentityrefBase, fmt.Sprintf("identityref %s must have at least one base statement", node.Name()))
			return
		}

		for _, base := range typeDef.Bases {
			v.resolveIdentityBase(base, node.Name())
		}
	}
}

func (v *Validator) resolveIdentityBase(base, nodeName string) {
	parts := strings.Split(base, ":")
	if len(parts) == 1 {
		// Local identity
		if _, ok := v.module.Identities[base]; !ok {
			v.compiler.addError(ErrIdentityrefBase, fmt.Sprintf("invalid identityref base '%s' in %s: identity not found locally", base, nodeName))
		}
	} else if len(parts) == 2 { // valid prefix usage
		prefix := parts[0]
		localName := parts[1]
		if modName, ok := v.compiler.imports[prefix]; ok && v.compiler.loader != nil {
			astMod, _ := v.compiler.loader.LoadAST(modName)
			found := false
			if astMod != nil {
				for _, stmt := range astMod.SubStatements() {
					if stmt.Keyword() == "identity" && stmt.Argument() == localName {
						found = true
						break
					}
				}
			}
			if !found {
				v.compiler.addError(ErrIdentityrefBase, fmt.Sprintf("invalid identityref base '%s' in %s: identity not found in imported module %s", base, nodeName, modName))
			}
		} else {
			v.compiler.addError(ErrIdentityrefBase, fmt.Sprintf("invalid identityref base '%s' in %s: unknown prefix %s", base, nodeName, prefix))
		}
	}
}

func (v *Validator) validateDefaultValues(nodes map[string]schema.Node) {
	for _, node := range nodes {
		if leaf, ok := node.(*schema.Leaf); ok {
			if leaf.Default != nil {
				v.checkTypeMatch(leaf.Name(), *leaf.Default, &leaf.Type)
			}
		}
		if len(node.GetChildren()) > 0 {
			v.validateDefaultValues(node.GetChildren())
		}
	}
}

func (v *Validator) checkTypeMatch(nodeName, val string, td *schema.TypeDefinition) {
	switch td.Name {
	case "boolean":
		if val != "true" && val != "false" {
			v.compiler.addError(ErrInvalidDefault, fmt.Sprintf("invalid default '%s' for boolean leaf %s", val, nodeName))
		}
	case "int8", "int16", "int32", "int64":
		// Basic parsing check
		if _, err := strconv.ParseInt(val, 10, 64); err != nil {
			v.compiler.addError(ErrInvalidDefault, fmt.Sprintf("invalid default '%s' for integer leaf %s", val, nodeName))
		}
	case "uint8", "uint16", "uint32", "uint64":
		if _, err := strconv.ParseUint(val, 10, 64); err != nil {
			v.compiler.addError(ErrInvalidDefault, fmt.Sprintf("invalid default '%s' for unsigned integer leaf %s", val, nodeName))
		}
	case "enumeration":
		if len(td.Enums) > 0 {
			found := false
			for _, e := range td.Enums {
				if e == val {
					found = true
					break
				}
			}
			if !found {
				v.compiler.addError(ErrInvalidDefault, fmt.Sprintf("invalid default '%s' for enum leaf %s", val, nodeName))
			}
		}
	}
}

func (v *Validator) validateXPathSyntax(nodes map[string]schema.Node) {
	for _, node := range nodes {
		// 1. Check When statement
		if base := node.GetBase(); base != nil {
			if base.When != nil {
				v.checkXPathSyntax(node.Name(), "when", *base.When)
			}
			for _, must := range base.Musts {
				v.checkXPathSyntax(node.Name(), "must", must)
			}
		}

		// 2. Check path in leafref
		if leaf, ok := node.(*schema.Leaf); ok && leaf.Type.Path != nil {
			v.checkXPathSyntax(node.Name(), "path", *leaf.Type.Path)
		} else if ll, ok := node.(*schema.LeafList); ok && ll.Type.Path != nil {
			v.checkXPathSyntax(node.Name(), "path", *ll.Type.Path)
		}

		if len(node.GetChildren()) > 0 {
			v.validateXPathSyntax(node.GetChildren())
		}
	}
}

// checkXPathSyntax performs basic sanity checks on XPath 1.0 expressions.
// It detects mismatched quotes or brackets as a rudimentary lint.
func (v *Validator) checkXPathSyntax(nodeName, stmtType, expr string) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		v.compiler.addError(ErrXPathSyntax, fmt.Sprintf("empty %s expression in node %s", stmtType, nodeName))
		return
	}

	singleQuote := 0
	doubleQuote := 0
	bracketLevel := 0
	parenLevel := 0

	for _, ch := range expr {
		switch ch {
		case '\'':
			if doubleQuote%2 == 0 {
				singleQuote++
			}
		case '"':
			if singleQuote%2 == 0 {
				doubleQuote++
			}
		case '[':
			if singleQuote%2 == 0 && doubleQuote%2 == 0 {
				bracketLevel++
			}
		case ']':
			if singleQuote%2 == 0 && doubleQuote%2 == 0 {
				bracketLevel--
			}
		case '(':
			if singleQuote%2 == 0 && doubleQuote%2 == 0 {
				parenLevel++
			}
		case ')':
			if singleQuote%2 == 0 && doubleQuote%2 == 0 {
				parenLevel--
			}
		}
	}

	if singleQuote%2 != 0 || doubleQuote%2 != 0 {
		v.compiler.addError(ErrXPathSyntax, fmt.Sprintf("mismatched quotes in %s expression '%s' for node %s", stmtType, expr, nodeName))
	}
	if bracketLevel != 0 || parenLevel != 0 {
		v.compiler.addError(ErrXPathSyntax, fmt.Sprintf("mismatched brackets or parentheses in %s expression '%s' for node %s", stmtType, expr, nodeName))
	}
}
