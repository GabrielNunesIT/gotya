// Package generator provides the interface for code generation.
package generator

import (
	"io"

	"github.com/gotya/gotya/schema"
)

// Generator defines the interface for creating source code from a compiled schema tree.
type Generator interface {
	// Generate takes a validated YANG module and writes the resulting output to the provided io.Writer.
	Generate(mod *schema.Module, w io.Writer) error

	// GenerateDevice takes a list of validated YANG modules, aggregates them into a single Device root struct,
	// and writes the resulting output to the provided io.Writer.
	GenerateDevice(modules []*schema.Module, w io.Writer) error
}
