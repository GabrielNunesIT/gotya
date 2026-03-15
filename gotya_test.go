package gotya_test

import (
	"errors"
	"os"
	"testing"

	"github.com/gotya/gotya"
	"github.com/stretchr/testify/require"
)

const validYANG = `
module test {
  yang-version 1;
  namespace "urn:test";
  prefix "t";
}
`

// TestParse_Error asserts that parsing invalid YANG returns an error that can be
// unwrapped as *gotya.ParseError.
func TestParse_Error(t *testing.T) {
	_, err := gotya.Parse("not valid yang")
	require.Error(t, err)

	var parseErr *gotya.ParseError
	require.True(t, errors.As(err, &parseErr), "expected *gotya.ParseError, got %T", err)
}

// TestParse_ErrorsAs asserts that the ParseError carries at least one Diagnostic with
// a non-zero Line field.
func TestParse_ErrorsAs(t *testing.T) {
	_, err := gotya.Parse("not valid yang")
	require.Error(t, err)

	var parseErr *gotya.ParseError
	require.True(t, errors.As(err, &parseErr))
	require.Greater(t, len(parseErr.Errors), 0, "expected at least one diagnostic")
	require.Greater(t, parseErr.Errors[0].Line, 0, "expected non-zero line number")
}

// TestParse_MultipleErrors asserts that parsing YANG with two distinct syntax errors
// returns a ParseError with at least two diagnostics.
func TestParse_MultipleErrors(t *testing.T) {
	// Two badly-formed statements inside a module body.
	badYANG := `
module multi-err {
  yang-version 1;
  namespace "urn:test";
  prefix "t";
  @@bad-statement-one;
  @@bad-statement-two;
}
`
	_, err := gotya.Parse(badYANG)
	require.Error(t, err)

	var parseErr *gotya.ParseError
	require.True(t, errors.As(err, &parseErr))
	require.GreaterOrEqual(t, len(parseErr.Errors), 2, "expected at least two diagnostics")
}

// TestCompile_OpaqueReturn asserts that Compile returns a slice whose element type
// exposes a Schema() method — without needing to import the schema package.
func TestCompile_OpaqueReturn(t *testing.T) {
	mods, err := gotya.Parse(validYANG)
	require.NoError(t, err)
	require.NotNil(t, mods)

	compiled, err := gotya.Compile([]*gotya.ASTModule{mods}, nil)
	require.NoError(t, err)
	require.Len(t, compiled, 1)

	// Schema() must be accessible at compile time without importing schema package.
	s := compiled[0].Schema()
	require.NotNil(t, s)
}

// TestASTModule_Name asserts that Parse returns an *ASTModule whose Name() method
// returns a non-empty string, confirming encapsulation (only Name() is called).
func TestASTModule_Name(t *testing.T) {
	mod, err := gotya.Parse(validYANG)
	require.NoError(t, err)
	require.NotNil(t, mod)

	name := mod.Name()
	require.NotEmpty(t, name)
}

// TestParseFile_FilledDiagnostic asserts two things:
//  1. ParseFile on a nonexistent path returns a plain I/O error (not a ParseError).
//  2. ParseFile on a real file containing invalid YANG returns a *gotya.ParseError
//     whose first Diagnostic has a non-empty File field.
func TestParseFile_FilledDiagnostic(t *testing.T) {
	// Case 1: nonexistent file — should be an I/O error, not a ParseError.
	_, err := gotya.ParseFile("nonexistent_file_that_does_not_exist.yang")
	require.Error(t, err)

	var parseErr *gotya.ParseError
	require.False(t, errors.As(err, &parseErr), "I/O error should not be wrapped as ParseError")

	// Case 2: real file with invalid YANG — should be a ParseError with File set.
	dir := t.TempDir()
	path := dir + "/invalid.yang"
	require.NoError(t, os.WriteFile(path, []byte("not valid yang"), 0o644))

	_, err = gotya.ParseFile(path)
	require.Error(t, err)
	require.True(t, errors.As(err, &parseErr), "expected *gotya.ParseError, got %T", err)
	require.NotEmpty(t, parseErr.Errors[0].File, "expected File field to be set")
}
