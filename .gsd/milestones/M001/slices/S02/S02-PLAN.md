# S02: Testing Infrastructure

**Goal:** Introduce typed error sentinels in the compiler package and migrate all string-matching
error assertions in compiler_test.
**Demo:** Introduce typed error sentinels in the compiler package and migrate all string-matching
error assertions in compiler_test.

## Must-Haves


## Tasks

- [x] **T01: 02-testing-infrastructure 01**
  - Introduce typed error sentinels in the compiler package and migrate all string-matching
error assertions in compiler_test.go to errors.Is / errors.As.

Purpose: Tests must not be coupled to error message wording. Refactoring a message must
not break a test. Sentinels provide stable, typed identities for error conditions.

Output:
- compiler/compiler.go — exported Err* vars, c.errors changed from []string to []error,
  addError signature changed to addError(sentinel error, msg string), Compile() returns
  errors.Join(c.errors...)
- compiler/compiler_test.go — all assert.Contains(t, err.Error(), ...) replaced with
  assert.ErrorIs(t, err, compiler.ErrXxx), table-driven tests updated to wantSentinel field
- [x] **T02: 02-testing-infrastructure 02**
  - Integrate go/format.Source() into the Go generator's GenerateDevice output path so that
syntactically invalid generated Go is caught at generation time and surfaces as an error
— never reaching the caller's io.Writer.

Purpose: Invalid generated Go must not silently reach the user's filesystem. Catching it
at the generation boundary gives a precise error with line/column info from the formatter.

Output:
- generator/golang/generator.go — GenerateDevice calls format.Source(buf.Bytes()) before
  w.Write; returns error if format fails
- generator/golang/generator_test.go — TestGoGenerator_FormatValidation test that asserts
  an error is returned (not written) when generation would produce bad syntax
- [x] **T03: 02-testing-infrastructure 03**
  - Convert the corpus pipeline driver in test/generate.go into a proper go test that runs
every YANG file in test/assets/yangs/ through the full parse→compile→generate pipeline
and asserts no panic and syntactically valid Go output.

Purpose: The corpus is a regression gate. Every future change that breaks pipeline output
on any of the 205 real-world YANG files must fail this test before it can merge.

Output:
- test/corpus_test.go — new file, package test, contains corpusLoader struct (copied and
  renamed from fileLoader in test/generate.go because generate.go has //go:build ignore
  and cannot be imported) and TestCorpus function with t.Run subtests per module.

## Files Likely Touched

- `compiler/compiler.go`
- `compiler/compiler_test.go`
- `generator/golang/generator.go`
- `generator/golang/generator_test.go`
- `test/corpus_test.go`
