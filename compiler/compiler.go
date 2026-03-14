// Package compiler provides the logic to convert an AST into a semantic schema tree.
package compiler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gotya/gotya/ast"
	"github.com/gotya/gotya/schema"
)

// ModuleLoader provides a way to load external modules and submodules.
type ModuleLoader interface {
	Load(name string) (*schema.Module, error)
	LoadAST(name string) (*ast.Module, error)
}

// Compiler orchestrates the conversion of an AST to a Schema tree.
type Compiler struct {
	errors             []string
	maxErrors          int                // maximum errors before truncation; default 100
	loader             ModuleLoader
	groupings          map[string]ast.Statement // stores module-level groupings
	typedefs           map[string]ast.Statement // stores module-level typedefs
	externalGroupStack []map[string]ast.Statement
	imports            map[string]string   // prefix to module name
	importStack        []map[string]string // lexical imports mapping corresponding to grouping origins
	supportedFeatures  map[string]bool     // optional list of features the server supports
	augments           []ast.Statement     // augments collected during compilation
}

// Options allows configuring the compiler behavior.
type Options struct {
	Loader            ModuleLoader
	SupportedFeatures []string
	MaxErrors         int // 0 means use default of 100
}

// New creates a new Compiler instance.
func New(opts *Options) *Compiler {
	c := &Compiler{
		errors:             make([]string, 0),
		maxErrors:          100,
		groupings:          make(map[string]ast.Statement),
		typedefs:           make(map[string]ast.Statement),
		externalGroupStack: make([]map[string]ast.Statement, 0),
		imports:            make(map[string]string),
		importStack:        make([]map[string]string, 0),
		supportedFeatures:  make(map[string]bool),
		augments:           make([]ast.Statement, 0),
	}
	if opts != nil {
		c.loader = opts.Loader
		for _, feature := range opts.SupportedFeatures {
			c.supportedFeatures[feature] = true
		}
		if opts.MaxErrors > 0 {
			c.maxErrors = opts.MaxErrors
		}
	}
	return c
}

// addError appends an error message, enforcing the maxErrors limit.
func (c *Compiler) addError(msg string) {
	if len(c.errors) > 0 && strings.HasPrefix(c.errors[len(c.errors)-1], "compilation stopped:") {
		return // already truncated; drop silently
	}
	c.errors = append(c.errors, msg)
	if len(c.errors) >= c.maxErrors {
		c.errors = append(c.errors, fmt.Sprintf("compilation stopped: too many errors (limit %d)", c.maxErrors))
	}
}

// Compile takes an AST module and converts it to a standard Schema Module.
func (c *Compiler) Compile(astMod *ast.Module) (*schema.Module, error) {
	if astMod == nil {
		return nil, errors.New("ast module cannot be nil")
	}

	mod := &schema.Module{
		Name:       astMod.Argument(),
		Nodes:      make(map[string]schema.Node),
		Features:   make(map[string]*schema.Feature),
		Identities: make(map[string]*schema.Identity),
	}

	// 0. Collect all statements, including submodules
	allStmts := make([]ast.Statement, 0, len(astMod.SubStatements()))
	allStmts = append(allStmts, astMod.SubStatements()...)

	// 1. Process module header definitions
	includedSubmodules := make(map[string]bool)
	for i := 0; i < len(allStmts); i++ {
		stmt := allStmts[i]
		switch stmt.Keyword() {
		case "namespace":
			mod.Namespace = stmt.Argument()
		case "prefix":
			mod.Prefix = stmt.Argument()
		case "import":
			moduleName := stmt.Argument()
			var prefix string
			for _, sub := range stmt.SubStatements() {
				if sub.Keyword() == "prefix" {
					prefix = sub.Argument()
				}
			}
			if prefix != "" {
				c.imports[prefix] = moduleName
			}
			if c.loader != nil {
				_, err := c.loader.Load(moduleName)
				if err != nil {
					c.addError(fmt.Sprintf("failed to import module %s: %v", moduleName, err))
				}
			}
		case "include":
			subName := stmt.Argument()
			if includedSubmodules[subName] {
				continue
			}
			includedSubmodules[subName] = true
			if c.loader != nil {
				subAST, err := c.loader.LoadAST(subName)
				if err != nil {
					c.addError(fmt.Sprintf("failed to include submodule %s: %v", subName, err))
				} else {
					// Merge submodule statements into our local processing slice, preserving the original AST
					allStmts = append(allStmts, subAST.SubStatements()...)
				}
			}
		}
	}

	// 2. Pre-process top-level groupings and typedefs
	c.groupings = make(map[string]ast.Statement)
	c.typedefs = make(map[string]ast.Statement)
	for _, stmt := range allStmts {
		if stmt.Keyword() == "grouping" {
			c.groupings[stmt.Argument()] = stmt
		} else if stmt.Keyword() == "typedef" {
			c.typedefs[stmt.Argument()] = stmt
		}
	}

	mod.Imports = c.imports
	c.importStack = append(c.importStack, c.imports)
	defer func() {
		c.importStack = c.importStack[:len(c.importStack)-1]
	}()

	// 3. Process data nodes (container, list, leaf, etc) and collect augments
	for _, stmt := range allStmts {
		if stmt.Keyword() == "uses" {
			c.resolveUses(stmt, nil, nil, mod, nil)
			continue
		}

		if stmt.Keyword() == "augment" {
			c.augments = append(c.augments, stmt)
			continue
		}

		node := c.compileDataNode(stmt, true)
		if node != nil {
			switch n := node.(type) {
			case *schema.Feature:
				mod.Features[n.Name()] = n
			case *schema.Deviation:
				mod.Deviations = append(mod.Deviations, n)
			case *schema.Identity:
				mod.Identities[n.Name()] = n
			default:
				if err := mod.AddNode(node); err != nil {
					c.addError(err.Error())
				}
			}
		}
	}

	// 4. Expose groupings mappings for cross-resolution imported modules
	mod.Groupings = c.groupings

	// 5. Process augments through multiple iterations to resolve chaining dependencies
	maxIter := len(c.augments) + 1
	iter := 0
	progress := true
	for progress {
		progress = false
		if iter >= maxIter {
			for _, aug := range c.augments {
				c.addError("augment target not found (loop cap reached): " + aug.Argument())
			}
			c.augments = nil
			break
		}
		iter++
		var pendingAugments []ast.Statement
		for _, aug := range c.augments {
			targetPath := aug.Argument()
			targetNode := c.findNode(mod, targetPath)
			if targetNode == nil {
				pendingAugments = append(pendingAugments, aug)
			} else {
				beforeKeys := make(map[string]bool)
				for k := range targetNode.GetChildren() {
					beforeKeys[k] = true
				}

				c.parseChildren(targetNode, aug.SubStatements(), nil)
				
				for k, child := range targetNode.GetChildren() {
					if !beforeKeys[k] {
						stampModuleName(map[string]schema.Node{k: child}, mod.Name)
					}
				}
				progress = true
			}
		}
		c.augments = pendingAugments
	}

	for _, aug := range c.augments {
		c.addError("augment target not found: "+aug.Argument())
	}

	// 5. Global validation mapping RFC constraints
	for _, node := range mod.Nodes {
		c.validate(node)
	}

	// 5. Semantic Validation
	validator := NewValidator(c, mod)
	if err := validator.Validate(); err != nil {
		c.addError(err.Error())
	}

	// 6. Feature Pruning
	if len(c.supportedFeatures) > 0 {
		c.pruneUnsupportedFeatures(mod.Nodes)
	}

	// 7. Apply Deviations
	c.applyDeviations(mod)

	// 8. Stamp each node with its originating module name for RFC 7951 codec support
	stampModuleName(mod.Nodes, mod.Name)

	if len(c.errors) > 0 {
		return mod, fmt.Errorf("compilation failed with %d errors:\n%s", len(c.errors), strings.Join(c.errors, "\n"))
	}
	return mod, nil
}

// stampModuleName recursively sets ModuleName on every node in the tree.
func stampModuleName(nodes map[string]schema.Node, moduleName string) {
	for _, node := range nodes {
		node.GetBase().ModuleName = moduleName
		if children := node.GetChildren(); len(children) > 0 {
			stampModuleName(children, moduleName)
		}
	}
}

func (c *Compiler) compileDataNode(stmt ast.Statement, parentConfig bool) schema.Node {
	configVal := parentConfig
	var musts []string
	var whenStmt *string
	var description, reference, status *string
	var ifFeatures []string
	for _, sub := range stmt.SubStatements() {
		if sub.Keyword() == "config" {
			configVal = sub.Argument() == "true"
		} else if sub.Keyword() == "must" {
			musts = append(musts, sub.Argument())
		} else if sub.Keyword() == "when" {
			val := sub.Argument()
			whenStmt = &val
		} else if sub.Keyword() == "description" {
			val := sub.Argument()
			description = &val
		} else if sub.Keyword() == "reference" {
			val := sub.Argument()
			reference = &val
		} else if sub.Keyword() == "status" {
			val := sub.Argument()
			status = &val
		} else if sub.Keyword() == "if-feature" {
			ifFeatures = append(ifFeatures, sub.Argument())
		}
	}

	var node schema.Node
	switch stmt.Keyword() {
	case "container":
		container := schema.NewContainer(stmt.Argument())
		container.SetConfig(configVal)
		container.Musts = musts
		container.When = whenStmt
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "presence" {
				val := sub.Argument()
				container.Presence = &val
			}
		}
		c.parseChildren(container, stmt.SubStatements(), nil)
		node = container
	case "list":
		list := schema.NewList(stmt.Argument())
		list.SetConfig(configVal)
		list.Musts = musts
		list.When = whenStmt
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "key" {
				list.Keys = strings.Fields(sub.Argument())
			} else if sub.Keyword() == "unique" {
				list.Unique = append(list.Unique, strings.Fields(sub.Argument()))
			} else if sub.Keyword() == "min-elements" {
				if val, err := strconv.ParseUint(sub.Argument(), 10, 32); err == nil {
					v := uint32(val)
					list.MinElements = &v
				}
			} else if sub.Keyword() == "max-elements" {
				if sub.Argument() == "unbounded" {
					list.MaxElements = nil
				} else if val, err := strconv.ParseUint(sub.Argument(), 10, 32); err == nil {
					v := uint32(val)
					list.MaxElements = &v
				}
			} else if sub.Keyword() == "ordered-by" {
				v := sub.Argument()
				list.OrderedBy = &v
			}
		}
		c.parseChildren(list, stmt.SubStatements(), nil)
		node = list
	case "leaf":
		var isMandatory bool
		var defValue, units *string
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "mandatory" {
				isMandatory = sub.Argument() == "true"
			} else if sub.Keyword() == "default" {
				val := sub.Argument()
				defValue = &val
			} else if sub.Keyword() == "units" {
				val := sub.Argument()
				units = &val
			}
		}
		leafType := c.getType(stmt.SubStatements(), nil)
		leaf := schema.NewLeaf(stmt.Argument(), &leafType)
		leaf.SetConfig(configVal)
		leaf.Musts = musts
		leaf.When = whenStmt
		leaf.Mandatory = isMandatory
		leaf.Default = defValue
		leaf.Units = units
		node = leaf
	case "leaf-list":
		leafType := c.getType(stmt.SubStatements(), nil)
		leafList := schema.NewLeafList(stmt.Argument(), &leafType)
		leafList.SetConfig(configVal)
		leafList.Musts = musts
		leafList.When = whenStmt
		var units *string
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "min-elements" {
				if val, err := strconv.ParseUint(sub.Argument(), 10, 32); err == nil {
					v := uint32(val)
					leafList.MinElements = &v
				}
			} else if sub.Keyword() == "max-elements" {
				if sub.Argument() == "unbounded" {
					leafList.MaxElements = nil
				} else if val, err := strconv.ParseUint(sub.Argument(), 10, 32); err == nil {
					v := uint32(val)
					leafList.MaxElements = &v
				}
			} else if sub.Keyword() == "ordered-by" {
				v := sub.Argument()
				leafList.OrderedBy = &v
			} else if sub.Keyword() == "units" {
				val := sub.Argument()
				units = &val
			}
		}
		leafList.Units = units
		node = leafList
	case "choice":
		choice := schema.NewChoice(stmt.Argument())
		choice.SetConfig(configVal)
		choice.Musts = musts
		choice.When = whenStmt
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "mandatory" {
				choice.Mandatory = sub.Argument() == "true"
			} else if sub.Keyword() == "default" {
				val := sub.Argument()
				choice.Default = &val
			}
		}
		c.parseChildren(choice, stmt.SubStatements(), nil)
		node = choice
	case "case":
		caseNode := schema.NewCase(stmt.Argument())
		caseNode.SetConfig(configVal)
		caseNode.Musts = musts
		caseNode.When = whenStmt
		c.parseChildren(caseNode, stmt.SubStatements(), nil)
		node = caseNode
	case "anyxml":
		anyxml := schema.NewAnyXML(stmt.Argument())
		anyxml.SetConfig(configVal)
		anyxml.Musts = musts
		anyxml.When = whenStmt
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "mandatory" {
				anyxml.Mandatory = sub.Argument() == "true"
			}
		}
		node = anyxml
	case "anydata":
		anydata := schema.NewAnyData(stmt.Argument())
		anydata.SetConfig(configVal)
		anydata.Musts = musts
		anydata.When = whenStmt
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "mandatory" {
				anydata.Mandatory = sub.Argument() == "true"
			}
		}
		node = anydata
	case "rpc":
		rpcNode := schema.NewRPC(stmt.Argument())
		rpcNode.Musts = musts
		c.parseChildren(rpcNode, stmt.SubStatements(), nil)
		if rpcNode.GetChildren()["input"] == nil {
			if err := rpcNode.AddChild(schema.NewInput()); err != nil {
				c.addError(err.Error())
			}
		}
		if rpcNode.GetChildren()["output"] == nil {
			if err := rpcNode.AddChild(schema.NewOutput()); err != nil {
				c.addError(err.Error())
			}
		}
		node = rpcNode
	case "action":
		actionNode := schema.NewAction(stmt.Argument())
		actionNode.Musts = musts
		c.parseChildren(actionNode, stmt.SubStatements(), nil)
		if actionNode.GetChildren()["input"] == nil {
			if err := actionNode.AddChild(schema.NewInput()); err != nil {
				c.addError(err.Error())
			}
		}
		if actionNode.GetChildren()["output"] == nil {
			if err := actionNode.AddChild(schema.NewOutput()); err != nil {
				c.addError(err.Error())
			}
		}
		node = actionNode
	case "notification":
		notif := schema.NewNotification(stmt.Argument())
		notif.Musts = musts
		c.parseChildren(notif, stmt.SubStatements(), nil)
		node = notif
	case "input":
		inputNode := schema.NewInput()
		inputNode.Musts = musts
		c.parseChildren(inputNode, stmt.SubStatements(), nil)
		node = inputNode
	case "output":
		outputNode := schema.NewOutput()
		outputNode.Musts = musts
		c.parseChildren(outputNode, stmt.SubStatements(), nil)
		node = outputNode
	case "feature":
		featureNode := schema.NewFeature(stmt.Argument())
		node = featureNode
	case "deviation":
		devNode := schema.NewDeviation(stmt.Argument())
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "deviate" {
				action := sub.Argument()
				devNode.Deviates[action] = append(devNode.Deviates[action], sub.SubStatements()...)
			}
		}
		node = devNode
	case "identity":
		idNode := schema.NewIdentity(stmt.Argument())
		for _, sub := range stmt.SubStatements() {
			if sub.Keyword() == "base" {
				idNode.Bases = append(idNode.Bases, sub.Argument())
			}
		}
		node = idNode
	}

	if node != nil {
		node.SetMetadata(description, reference, status, ifFeatures)
	}

	return node
}

func (c *Compiler) validate(node schema.Node) {
	switch n := node.(type) {
	case *schema.List:
		if n.Config() && len(n.Keys) == 0 {
			c.addError("list "+n.Name()+" must have at least one key if it is configuration data")
		}
		for _, keyName := range n.Keys {
			childNode, exists := n.GetChildren()[keyName]
			if !exists {
				c.addError(fmt.Sprintf("key '%s' not found in list '%s'", keyName, n.Name()))
				continue
			}
			if _, isLeaf := childNode.(*schema.Leaf); !isLeaf {
				c.addError(fmt.Sprintf("key '%s' in list '%s' must be a leaf", keyName, n.Name()))
			}
		}
	case *schema.Leaf:
		c.validateType(n.Name(), &n.Type)
		if n.Mandatory && n.Default != nil {
			c.addError(fmt.Sprintf("leaf '%s' is mandatory and cannot have a default value", n.Name()))
		}
	case *schema.LeafList:
		c.validateType(n.Name(), &n.Type)
	}

	for _, child := range node.GetChildren() {
		if !node.Config() && child.Config() {
			c.addError(fmt.Sprintf("node %s has 'config true' but its parent %s has 'config false'", child.Name(), node.Name()))
		}
		c.validate(child)
	}
}

func (c *Compiler) validateType(nodeName string, td *schema.TypeDefinition) {
	if len(td.Length) > 0 {
		if td.Name != "string" && td.Name != "binary" {
			c.addError(fmt.Sprintf("type %s for node %s cannot have length restrictions", td.Name, nodeName))
		}
	}
	if len(td.Pattern) > 0 {
		if td.Name != "string" {
			c.addError(fmt.Sprintf("type %s for node %s cannot have pattern restrictions", td.Name, nodeName))
		}
	}
	if len(td.Range) > 0 {
		switch td.Name {
		case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "decimal64":
			// Valid
		default:
			c.addError(fmt.Sprintf("type %s for node %s cannot have range restrictions", td.Name, nodeName))
		}
	}
}

// pruneUnsupportedFeatures recursively removes nodes that have unmet if-feature conditions.
func (c *Compiler) pruneUnsupportedFeatures(nodes map[string]schema.Node) {
	for name, node := range nodes {
		base := node.GetBase()
		if base != nil && len(base.IfFeatures) > 0 {
			supported := true
			for _, expr := range base.IfFeatures {
				if !c.evaluateIfFeature(expr) {
					supported = false
					break
				}
			}
			if !supported {
				delete(nodes, name)
				continue
			}
		}
		if len(node.GetChildren()) > 0 {
			c.pruneUnsupportedFeatures(node.GetChildren())
		}
	}
}

// evaluateIfFeature evaluates a boolean expression of features.
// Supports simple names, "not X", "X or Y", "X and Y", and parentheses.
// This is a simplified evaluator for YANG 1.1 if-feature syntax.
func (c *Compiler) evaluateIfFeature(expr string) bool {
	expr = strings.TrimSpace(expr)

	// Basic recursive evaluation (shunting-yard or simple recursive descent would be better for full compliance,
	// but this covers the basics: "X", "not X", "X or Y", "X and Y").

	// Remove outer parentheses if present
	for strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		expr = strings.TrimSpace(expr[1 : len(expr)-1])
	}

	// Handle 'or'
	if parts := strings.Split(expr, " or "); len(parts) > 1 {
		for _, part := range parts {
			if c.evaluateIfFeature(part) {
				return true
			}
		}
		return false
	}

	// Handle 'and'
	if parts := strings.Split(expr, " and "); len(parts) > 1 {
		for _, part := range parts {
			if !c.evaluateIfFeature(part) {
				return false
			}
		}
		return true
	}

	// Handle 'not'
	if strings.HasPrefix(expr, "not ") {
		return !c.evaluateIfFeature(strings.TrimPrefix(expr, "not "))
	}

	// Leaf term: check if the feature is in the supported set
	// Note: in a real implementation we would check the prefix against imports to find the right module's feature list.
	// For this test, we assume features are globally mapped.
	return c.supportedFeatures[expr]
}

func (c *Compiler) applyDeviations(mod *schema.Module) {
	for _, dev := range mod.Deviations {
		target := c.findNode(mod, dev.Name())
		if target == nil {
			c.addError(fmt.Sprintf("deviation target node %s not found", dev.Name()))
			continue
		}

		for action, stmts := range dev.Deviates {
			switch action {
			case "not-supported":
				c.removeNode(mod, target)
			case "add", "replace":
				for _, stmt := range stmts {
					switch stmt.Keyword() {
					case "config":
						target.SetConfig(stmt.Argument() == "true")
					case "mandatory":
						switch n := target.(type) {
						case *schema.Leaf:
							n.Mandatory = stmt.Argument() == "true"
						case *schema.Choice:
							n.Mandatory = stmt.Argument() == "true"
						case *schema.AnyXML:
							n.Mandatory = stmt.Argument() == "true"
						case *schema.AnyData:
							n.Mandatory = stmt.Argument() == "true"
						}
					case "default":
						val := stmt.Argument()
						switch n := target.(type) {
						case *schema.Leaf:
							n.Default = &val
						case *schema.Choice:
							n.Default = &val
						}
					case "fraction-digits":
						if val, err := strconv.ParseUint(stmt.Argument(), 10, 8); err == nil {
							v := uint8(val)
							switch n := target.(type) {
							case *schema.Leaf:
								n.Type.FractionDigits = &v
							case *schema.LeafList:
								n.Type.FractionDigits = &v
							}
						}
					case "path":
						val := stmt.Argument()
						switch n := target.(type) {
						case *schema.Leaf:
							n.Type.Path = &val
						case *schema.LeafList:
							n.Type.Path = &val
						}
					}
				}
			case "delete":
				for _, stmt := range stmts {
					if stmt.Keyword() == "default" {
						switch n := target.(type) {
						case *schema.Leaf:
							n.Default = nil
						case *schema.Choice:
							n.Default = nil
						}
					}
				}
			}
		}
	}
}

func (c *Compiler) removeNode(mod *schema.Module, target schema.Node) {
	// Root level
	if _, ok := mod.Nodes[target.Name()]; ok {
		delete(mod.Nodes, target.Name())
		return
	}
	// Recursive search and destroy
	c.unparentNode(mod.Nodes, target)
}

func (c *Compiler) unparentNode(nodes map[string]schema.Node, target schema.Node) {
	for _, node := range nodes {
		if node.GetChildren() != nil {
			if _, ok := node.GetChildren()[target.Name()]; ok {
				delete(node.GetChildren(), target.Name())
				return
			}
			c.unparentNode(node.GetChildren(), target)
		}
	}
}

func (c *Compiler) parseChildren(parent schema.Node, stmts []ast.Statement, visited map[string]bool) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	// First collect groupings (lazy evaluation handled by AST storage)
	groupings := make(map[string]ast.Statement)
	for _, sub := range stmts {
		if sub.Keyword() == "grouping" {
			groupings[sub.Argument()] = sub
		}
	}

	c.externalGroupStack = append(c.externalGroupStack, groupings)
	defer func() {
		c.externalGroupStack = c.externalGroupStack[:len(c.externalGroupStack)-1]
	}()

	for _, sub := range stmts {
		if sub.Keyword() == "uses" {
			c.resolveUses(sub, groupings, visited, nil, parent)
			continue
		}

		if sub.Keyword() == "augment" {
			c.augments = append(c.augments, sub)
			continue
		}

		child := c.compileDataNode(sub, parent.Config())
		if child != nil {
			if _, isChoice := parent.(*schema.Choice); isChoice {
				if _, isCase := child.(*schema.Case); !isCase {
					// implicitly create a short-hand case
					caseNode := schema.NewCase(child.Name())
					caseNode.SetConfig(child.Config())
					if err := caseNode.AddChild(child); err != nil {
					c.addError(err.Error())
				}
					child = caseNode
				}
			}

			if err := parent.AddChild(child); err != nil {
				c.addError(err.Error())
			}
		}
	}
}

// findNode resolves a path expression to a Schema Node
func (c *Compiler) findNode(mod *schema.Module, path string) schema.Node {
	parts := strings.Split(path, "/")
	var current schema.Node

	for _, part := range parts {
		if part == "" {
			continue
		}

		name := part
		var prefix string
		if idx := strings.Index(name, ":"); idx != -1 {
			prefix = name[:idx]
			name = name[idx+1:]
		}

		if current == nil {
			if prefix != "" && prefix != mod.Prefix {
				var currImports map[string]string
				if len(c.importStack) > 0 {
					currImports = c.importStack[len(c.importStack)-1]
				} else {
					currImports = c.imports
				}
				if modName, ok := currImports[prefix]; ok && c.loader != nil {
					if targetMod, err := c.loader.Load(modName); err == nil && targetMod != nil {
						current = targetMod.Nodes[name]
					}
				}
			} else {
				current = mod.Nodes[name]
			}
		} else {
			current = current.GetChildren()[name]
		}

		if current == nil {
			return nil
		}
	}
	return current
}

// getType looks for a 'type' substatement and extracts its definition with restrictions.
// visited tracks typedef names being resolved to detect circular references; pass nil on first call.
func (c *Compiler) getType(stmts []ast.Statement, visited map[string]bool) schema.TypeDefinition {
	var td schema.TypeDefinition
	for _, stmt := range stmts {
		if stmt.Keyword() == "type" {
			td.Name = stmt.Argument()

			// Check if this type name references a known typedef
			if typedefAST, ok := c.typedefs[td.Name]; ok {
				td.TypedefName = td.Name
				// Cycle detection for typedef resolution
				if visited == nil {
					visited = make(map[string]bool)
				}
				if visited[td.Name] {
					c.addError("circular typedef detected: " + td.Name)
					return td
				}
				visited[td.Name] = true
				defer delete(visited, td.Name)
				// Resolve the underlying type from the typedef
				resolved := c.getType(typedefAST.SubStatements(), visited)
				// Merge: keep the typedef name but use the resolved base type and its constraints
				td.Name = resolved.Name
				if len(resolved.Enums) > 0 {
					td.Enums = resolved.Enums
				}
				if len(resolved.Bits) > 0 {
					td.Bits = resolved.Bits
				}
				if len(resolved.Range) > 0 && len(td.Range) == 0 {
					td.Range = resolved.Range
				}
				if len(resolved.Length) > 0 && len(td.Length) == 0 {
					td.Length = resolved.Length
				}
				if len(resolved.Pattern) > 0 && len(td.Pattern) == 0 {
					td.Pattern = resolved.Pattern
				}
				if len(resolved.Members) > 0 {
					td.Members = resolved.Members
				}
				if resolved.Path != nil && td.Path == nil {
					td.Path = resolved.Path
				}
				if resolved.FractionDigits != nil && td.FractionDigits == nil {
					td.FractionDigits = resolved.FractionDigits
				}
				if len(resolved.Bases) > 0 {
					td.Bases = resolved.Bases
				}
			}

			// Process inline sub-statements (may override/extend typedef constraints)
			for _, sub := range stmt.SubStatements() {
				switch sub.Keyword() {
				case "range":
					td.Range = append(td.Range, sub.Argument())
				case "length":
					td.Length = append(td.Length, sub.Argument())
				case "pattern":
					td.Pattern = append(td.Pattern, sub.Argument())
				case "enum":
					td.Enums = append(td.Enums, sub.Argument())
				case "fraction-digits":
					if val, err := strconv.ParseUint(sub.Argument(), 10, 8); err == nil {
						v := uint8(val)
						td.FractionDigits = &v
					}
				case "base":
					td.Bases = append(td.Bases, sub.Argument())
				case "bit":
					td.Bits = append(td.Bits, sub.Argument())
				case "require-instance":
					v := sub.Argument() == "true"
					td.RequireInstance = &v
				case "path":
					val := sub.Argument()
					td.Path = &val
				case "type":
					// union member type — recurse to capture sub-constraints
					member := c.getType([]ast.Statement{sub}, nil)
					td.Members = append(td.Members, member)
				}
			}
			break
		}
	}
	if td.Name == "" {
		td.Name = "unknown"
	}
	return td
}

func (c *Compiler) resolveUses(stmt ast.Statement, localGroupings map[string]ast.Statement, visited map[string]bool, mod *schema.Module, parent schema.Node) {
	groupName := stmt.Argument()
	var grpAST ast.Statement

	if visited == nil {
		visited = make(map[string]bool)
	}
	if visited[groupName] {
		c.addError("circular dependency detected in uses: "+groupName)
		return
	}
	visited[groupName] = true
	defer delete(visited, groupName)

	var pushed bool
	if strings.Contains(groupName, ":") {
		parts := strings.Split(groupName, ":")
		prefix, localName := parts[0], parts[1]
		var currImports map[string]string
		if len(c.importStack) > 0 {
			currImports = c.importStack[len(c.importStack)-1]
		} else {
			currImports = c.imports
		}
		if modName, ok := currImports[prefix]; ok && c.loader != nil {
			targetMod, _ := c.loader.Load(modName)
			if targetMod != nil {
				if gst, ok := targetMod.Groupings[localName]; ok {
					grpAST = gst
					c.externalGroupStack = append(c.externalGroupStack, targetMod.Groupings)
					c.importStack = append(c.importStack, targetMod.Imports)
					pushed = true
				}
			}
		}
	}

	if grpAST == nil {
		searchName := groupName
		if idx := strings.Index(groupName, ":"); idx != -1 {
			searchName = groupName[idx+1:]
		}

		if grp, ok := localGroupings[searchName]; ok {
			grpAST = grp
		} else if grp, ok := c.groupings[searchName]; ok {
			grpAST = grp
		}

		if grpAST == nil {
			for i := len(c.externalGroupStack) - 1; i >= 0; i-- {
				if grp, ok := c.externalGroupStack[i][searchName]; ok {
					grpAST = grp
					break
				}
			}
		}
	}

	if grpAST == nil {
		c.addError("grouping not found: "+groupName)
		return
	}

	temp := schema.NewContainer("temp")
	pConfig := true
	if parent != nil {
		pConfig = parent.Config()
	}
	temp.SetConfig(pConfig)

	c.parseChildren(temp, grpAST.SubStatements(), visited)

	if pushed {
		c.externalGroupStack = c.externalGroupStack[:len(c.externalGroupStack)-1]
		c.importStack = c.importStack[:len(c.importStack)-1]
	}

	for _, sub := range stmt.SubStatements() {
		if sub.Keyword() == "refine" {
			targetPath := sub.Argument()
			targetNode := c.findNodeFrom(temp, targetPath)
			if targetNode != nil {
				c.applyRefine(targetNode, sub)
			} else {
				c.addError("refine target not found: "+targetPath)
			}
		} else if sub.Keyword() == "augment" {
			targetPath := sub.Argument()
			targetNode := c.findNodeFrom(temp, targetPath)
			if targetNode != nil {
				c.parseChildren(targetNode, sub.SubStatements(), visited)
			} else {
				c.addError("uses augment target not found: "+targetPath)
			}
		}
	}

	for _, child := range temp.GetChildren() {
		if mod != nil {
			if err := mod.AddNode(child); err != nil {
				c.addError(err.Error())
			}
		} else if parent != nil {
			if err := parent.AddChild(child); err != nil {
				c.addError(err.Error())
			}
		}
	}
}

func (c *Compiler) findNodeFrom(start schema.Node, path string) schema.Node {
	parts := strings.Split(path, "/")
	current := start
	for _, part := range parts {
		if part == "" {
			continue
		}
		name := part
		if idx := strings.Index(name, ":"); idx != -1 {
			name = name[idx+1:]
		}
		current = current.GetChildren()[name]
		if current == nil {
			return nil
		}
	}
	return current
}

func (c *Compiler) applyRefine(node schema.Node, refineStmt ast.Statement) {
	var bn *schema.BaseNode
	switch n := node.(type) {
	case *schema.Container:
		bn = n.BaseNode
	case *schema.List:
		bn = n.BaseNode
	case *schema.Leaf:
		bn = n.BaseNode
	case *schema.LeafList:
		bn = n.BaseNode
	case *schema.Choice:
		bn = n.BaseNode
	case *schema.Case:
		bn = n.BaseNode
	case *schema.AnyXML:
		bn = n.BaseNode
	case *schema.AnyData:
		bn = n.BaseNode
	}

	if bn == nil {
		return
	}

	for _, sub := range refineStmt.SubStatements() {
		switch sub.Keyword() {
		case "description":
			val := sub.Argument()
			bn.Description = &val
		case "reference":
			val := sub.Argument()
			bn.Reference = &val
		case "config":
			bn.IsConfig = sub.Argument() == "true"
		case "mandatory":
			isMand := sub.Argument() == "true"
			switch n := node.(type) {
			case *schema.Leaf:
				n.Mandatory = isMand
			case *schema.Choice:
				n.Mandatory = isMand
			case *schema.AnyXML:
				n.Mandatory = isMand
			case *schema.AnyData:
				n.Mandatory = isMand
			}
		case "presence":
			val := sub.Argument()
			if cont, ok := node.(*schema.Container); ok {
				cont.Presence = &val
			}
		case "default":
			val := sub.Argument()
			if leaf, ok := node.(*schema.Leaf); ok {
				leaf.Default = &val
			} else if choice, ok := node.(*schema.Choice); ok {
				choice.Default = &val
			}
		case "min-elements":
			if val, err := strconv.ParseUint(sub.Argument(), 10, 32); err == nil {
				v := uint32(val)
				if lst, ok := node.(*schema.List); ok {
					lst.MinElements = &v
				} else if ll, ok := node.(*schema.LeafList); ok {
					ll.MinElements = &v
				}
			}
		case "max-elements":
			var vPtr *uint32
			if sub.Argument() != "unbounded" {
				if val, err := strconv.ParseUint(sub.Argument(), 10, 32); err == nil {
					v := uint32(val)
					vPtr = &v
				}
			}
			if lst, ok := node.(*schema.List); ok {
				lst.MaxElements = vPtr
			} else if ll, ok := node.(*schema.LeafList); ok {
				ll.MaxElements = vPtr
			}
		}
	}
}
