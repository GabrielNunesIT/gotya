# Proto Generator — RFC 7950 Coverage

This document tracks which RFC 7950 §7 YANG statement types are supported by the gotya Protobuf generator (`generator/protobuf`). Every statement in RFC 7950 §7 is listed with its Proto mapping and support status.

**Status values:**
- `supported` — The generator handles this statement and emits correct Protobuf output.
- `unsupported` — The statement is present in the schema but the generator does not emit a typed Protobuf representation for it.
- `out-of-scope` — The statement is resolved or consumed before the generator runs (e.g., by the compiler), or the concept does not map to Protobuf output in v1.

## RFC 7950 §7 Statement Coverage

| YANG Statement | RFC Section | Proto Mapping | Status | Notes |
|---|---|---|---|---|
| module | 7.1 | package + top-level message | supported | Module name → proto package + CamelCase message |
| submodule | 7.2 | (merged by compiler) | out-of-scope | Compiler resolves includes; generator sees flat module |
| typedef | 7.3 | resolved to primitive | supported | Typedef expansion at compiler level |
| type | 7.4 | scalar proto type | supported | mapYANGTypeToProto() handles string/int/bool/bytes etc. |
| container | 7.5 | message | supported | Recursive message generation |
| leaf | 7.6 | scalar field | supported | CEL constraints via buildValidateOptions |
| leaf-list | 7.7 | repeated scalar field | supported | `repeated` prefix |
| list | 7.8 | repeated message field | supported | `repeated MessageType field` |
| choice | 7.9 | oneof | supported | Case children as CaseMessage types |
| anydata | 7.10 | google.protobuf.Any | supported | PBGEN-02; conditional import |
| anyxml | 7.11 | google.protobuf.Any | supported | PBGEN-02; same as anydata |
| grouping | 7.12 | (resolved by compiler) | out-of-scope | Compiler expands groupings before generator |
| uses | 7.13 | (resolved by compiler) | out-of-scope | Compiler expands uses before generator |
| rpc | 7.14 | service rpc + Input/Output messages | supported | PBGEN-03 |
| action | 7.15 | service rpc + Input/Output messages | supported | PBGEN-03 |
| notification | 7.16 | standalone message | supported | PBGEN-03; not in service block |
| augment | 7.17 | (resolved by compiler) | out-of-scope | Augment application at compiler level |
| identity | 7.18 | string field | unsupported | Identityref leaf emits string; no typed enum for identity hierarchy |
| extension | 7.19 | ignored | out-of-scope | Extension statements not in schema |
| feature / if-feature | 7.20.1–7.20.2 | ignored | out-of-scope | Feature guards not evaluated; nodes always emitted |
| deviation | 7.20.3 | not applied | out-of-scope | Deviation pre-pass not in proto generator (v1) |
| config | 7.21.1 | ignored | out-of-scope | Config/state split not emitted |
| status | 7.21.2 | SkipDeprecated/SkipObsolete options | supported | shouldSkip() filters deprecated/obsolete |
| description | 7.21.3 | // comment (AddAnnotations option) | supported | AddAnnotations=true emits description as comment |
| reference | 7.21.4 | ignored | out-of-scope | Reference statement not emitted |
| when | 7.21.5 | ignored | out-of-scope | XPath conditions not evaluatable in proto |

## Summary

| Status | Count |
|---|---|
| supported | 13 |
| unsupported | 1 |
| out-of-scope | 12 |
| **Total** | **26** |
