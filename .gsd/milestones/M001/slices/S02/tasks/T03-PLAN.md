# T03: 02-testing-infrastructure 03

**Slice:** S02 — **Milestone:** M001

## Description

Convert the corpus pipeline driver in test/generate.go into a proper go test that runs
every YANG file in test/assets/yangs/ through the full parse→compile→generate pipeline
and asserts no panic and syntactically valid Go output.

Purpose: The corpus is a regression gate. Every future change that breaks pipeline output
on any of the 205 real-world YANG files must fail this test before it can merge.

Output:
- test/corpus_test.go — new file, package test, contains corpusLoader struct (copied and
  renamed from fileLoader in test/generate.go because generate.go has //go:build ignore
  and cannot be imported) and TestCorpus function with t.Run subtests per module.

## Must-Haves

- [ ] "go test ./test/... -run TestCorpus completes without any panic"
- [ ] "TestCorpus reports a subtest for each YANG module in test/assets/yangs/"
- [ ] "TestCorpus fails if generated Go output does not pass go/format.Source()"
- [ ] "TestCorpus does NOT fail when a YANG module produces a compiler error (only logs it)"
- [ ] "go test ./test/... passes with the race detector enabled"

## Files

- `test/corpus_test.go`
