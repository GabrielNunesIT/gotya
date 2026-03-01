// Package schema provides semantic data tree models for compiled YANG modules.
package schema

import (
	"fmt"

	"github.com/gotya/gotya/domain/ast"
)

// Node is the interface implemented by all schema data// Node represents any node in the schema tree.
type Node interface {
	Name() string
	Config() bool
	AddChild(Node) error
	GetChildren() map[string]Node
	SetConfig(c bool)
	SetMetadata(description, reference, status *string, ifFeatures []string)
	GetBase() *BaseNode
}

// BaseNode provides standard data and methods for all schema nodes.
type BaseNode struct {
	NodeName    string
	IsConfig    bool
	Children    map[string]Node
	Order       int // Tracks the definition order in its parent
	Musts       []string
	When        *string
	Description *string
	Reference   *string
	Status      *string
	IfFeatures  []string
}

// Name returns the node's name.
func (b *BaseNode) Name() string { return b.NodeName }

// Config returns whether the node represents configuration data.
func (b *BaseNode) Config() bool { return b.IsConfig }

// SetConfig sets the configuration data flag.
func (b *BaseNode) SetConfig(config bool) { b.IsConfig = config }

// AddChild adds a child node to this node.
func (b *BaseNode) AddChild(child Node) error {
	if b.Children == nil {
		b.Children = make(map[string]Node)
	}
	if _, exists := b.Children[child.Name()]; exists {
		return fmt.Errorf("duplicate identifier '%s'", child.Name())
	}
	child.GetBase().Order = len(b.Children)
	b.Children[child.Name()] = child
	return nil
}

// GetChildren returns the child nodes.
func (b *BaseNode) GetChildren() map[string]Node {
	return b.Children
}

// SetMetadata sets standard YANG metadata on the node.
func (b *BaseNode) SetMetadata(description, reference, status *string, ifFeatures []string) {
	b.Description = description
	b.Reference = reference
	b.Status = status
	b.IfFeatures = ifFeatures
}

// GetBase returns the underlying BaseNode.
func (b *BaseNode) GetBase() *BaseNode {
	return b
}

// Module represents a compiled and resolved YANG module.
type Module struct {
	Name       string
	Namespace  string
	Prefix     string
	Nodes      map[string]Node // Data nodes at the top level
	Features   map[string]*Feature
	Deviations []*Deviation
	Identities map[string]*Identity
	Groupings  map[string]ast.Statement // Exported groupings for cross-resource uses resolution
	Imports    map[string]string        // Original imports mapping
}

// AddNode adds a top-level node to the module.
func (m *Module) AddNode(node Node) error {
	if m.Nodes == nil {
		m.Nodes = make(map[string]Node)
	}
	if _, exists := m.Nodes[node.Name()]; exists {
		return fmt.Errorf("duplicate identifier '%s' at module level", node.Name())
	}
	node.GetBase().Order = len(m.Nodes)
	m.Nodes[node.Name()] = node
	return nil
}

// Container is a collection of related nodes.
type Container struct {
	*BaseNode
	Presence *string
}

// List represents an array of structural data.
type List struct {
	*BaseNode
	Keys        []string
	Unique      [][]string
	MinElements *uint32
	MaxElements *uint32
	OrderedBy   *string
}

// TypeDefinition represents a YANG type, potentially with restrictions.
type TypeDefinition struct {
	Name            string
	Range           []string
	Length          []string
	Pattern         []string
	Enums           []string
	FractionDigits  *uint8
	Bases           []string
	Bits            []string
	RequireInstance *bool
	Path            *string
	// Members holds the ordered member types for a union type (RFC 7950 §7.4).
	Members []TypeDefinition
}

// Leaf represents a singular leaf node containing data.
type Leaf struct {
	*BaseNode
	Type      TypeDefinition
	Mandatory bool
	Default   *string
	Units     *string
}

// LeafList represents an array of leaf values.
type LeafList struct {
	*BaseNode
	Type        TypeDefinition
	MinElements *uint32
	MaxElements *uint32
	OrderedBy   *string
	Units       *string
}

// Choice represents a choice of alternatives.
type Choice struct {
	*BaseNode
	Mandatory bool
	Default   *string
}

// Case represents a specific choice alternative.
type Case struct {
	*BaseNode
}

// AnyXML represents an unconstrained XML node.
type AnyXML struct {
	*BaseNode
	Mandatory bool
}

// AnyData represents an unconstrained data node (YANG 1.1).
type AnyData struct {
	*BaseNode
	Mandatory bool
}

// NewContainer creates a new Container.
func NewContainer(name string) *Container {
	return &Container{BaseNode: &BaseNode{NodeName: name, IsConfig: true, Children: make(map[string]Node)}}
}

// NewList creates a new List.
func NewList(name string) *List {
	return &List{BaseNode: &BaseNode{NodeName: name, IsConfig: true, Children: make(map[string]Node)}}
}

// NewLeaf creates a new Leaf.
func NewLeaf(name string, typeDef *TypeDefinition) *Leaf {
	return &Leaf{BaseNode: &BaseNode{NodeName: name, IsConfig: true}, Type: *typeDef}
}

// NewLeafList creates a new LeafList.
func NewLeafList(name string, typeDef *TypeDefinition) *LeafList {
	return &LeafList{BaseNode: &BaseNode{NodeName: name, IsConfig: true}, Type: *typeDef}
}

// NewChoice creates a new Choice.
func NewChoice(name string) *Choice {
	return &Choice{BaseNode: &BaseNode{NodeName: name, IsConfig: true, Children: make(map[string]Node)}}
}

// NewCase creates a new Case.
func NewCase(name string) *Case {
	return &Case{BaseNode: &BaseNode{NodeName: name, IsConfig: true, Children: make(map[string]Node)}}
}

// NewAnyXML creates a new AnyXML.
func NewAnyXML(name string) *AnyXML {
	return &AnyXML{BaseNode: &BaseNode{NodeName: name, IsConfig: true}}
}

// NewAnyData creates a new AnyData.
func NewAnyData(name string) *AnyData {
	return &AnyData{BaseNode: &BaseNode{NodeName: name, IsConfig: true}}
}

// RPC represents an rpc operation.
type RPC struct {
	*BaseNode
}

// Action represents an action operation linked to a node.
type Action struct {
	*BaseNode
}

// Notification represents a notification message.
type Notification struct {
	*BaseNode
}

// Input represents the input of an operation.
type Input struct {
	*BaseNode
}

// Output represents the output of an operation.
type Output struct {
	*BaseNode
}

// NewRPC creates a new RPC.
func NewRPC(name string) *RPC {
	return &RPC{BaseNode: &BaseNode{NodeName: name, IsConfig: false, Children: make(map[string]Node)}}
}

// NewAction creates a new Action.
func NewAction(name string) *Action {
	return &Action{BaseNode: &BaseNode{NodeName: name, IsConfig: false, Children: make(map[string]Node)}}
}

// NewNotification creates a new Notification.
func NewNotification(name string) *Notification {
	return &Notification{BaseNode: &BaseNode{NodeName: name, IsConfig: false, Children: make(map[string]Node)}}
}

// NewInput creates a new Input.
func NewInput() *Input {
	return &Input{BaseNode: &BaseNode{NodeName: "input", IsConfig: false, Children: make(map[string]Node)}}
}

// NewOutput creates a new Output.
func NewOutput() *Output {
	return &Output{BaseNode: &BaseNode{NodeName: "output", IsConfig: false, Children: make(map[string]Node)}}
}

// Feature represents a module feature.
type Feature struct {
	*BaseNode
}

// NewFeature creates a new Feature.
func NewFeature(name string) *Feature {
	return &Feature{BaseNode: &BaseNode{NodeName: name, IsConfig: false, Children: make(map[string]Node)}}
}

// Deviation represents a module deviation.
type Deviation struct {
	*BaseNode
	Deviates map[string][]ast.Statement // e.g. "add" -> []ast.Statement (the properties to add)
}

// NewDeviation creates a new Deviation.
func NewDeviation(name string) *Deviation {
	return &Deviation{
		BaseNode: &BaseNode{NodeName: name, IsConfig: false, Children: make(map[string]Node)},
		Deviates: make(map[string][]ast.Statement),
	}
}

// Identity represents a globally unique identifier.
type Identity struct {
	*BaseNode
	Bases []string
}

// NewIdentity creates a new Identity.
func NewIdentity(name string) *Identity {
	return &Identity{BaseNode: &BaseNode{NodeName: name, IsConfig: false, Children: make(map[string]Node)}}
}
