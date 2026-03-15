# T03: 03-go-generator 03

**Slice:** S03 — **Milestone:** M001

## Description

Implement GOGEN-03: emit typed Go structs for rpc, action, and notification YANG statements.
Turn TestRPC, TestAction, TestNotification from RED to GREEN.

Purpose: Callers building gRPC or REST APIs from YANG need typed input/output types they can
instantiate and pass around — not just config/state data structs.

Output:
- generator/golang/generator.go — three additions:
  1. hasValidNodes extended to return true for RPC/Action/Notification
  2. generateNode handles *schema.RPC, *schema.Action, *schema.Notification
  3. GenerateDevice post-loop pass emits RPC/Action/Notification structs once per module (NOT
     inside the Config+State loop)
- generator/golang/generator_rpc_test.go — stubs replaced with real assertions

## Must-Haves

- [ ] "A YANG module with an rpc statement generates XxxInput and XxxOutput Go structs as top-level named types"
- [ ] "A YANG module with an action statement generates XxxInput and XxxOutput Go structs"
- [ ] "A YANG module with a notification statement generates a single flat XxxNotification Go struct"
- [ ] "RPC/Action/Notification structs are NOT duplicated by the Config+State loop — they appear exactly once"
- [ ] "TestRPC, TestAction, TestNotification are GREEN"

## Files

- `generator/golang/generator.go`
- `generator/golang/generator_rpc_test.go`
