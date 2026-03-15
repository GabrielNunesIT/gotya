# T03: 04-protobuf-generator 03

**Slice:** S04 — **Milestone:** M001

## Description

Implement PBGEN-03: rpc and action YANG statements emit Protobuf service blocks (one per module) with typed Input/Output messages; notification statements emit standalone messages.

Purpose: Enables gRPC service stub generation from YANG RPC definitions — the primary use case for proto output in network management.
Output: Modified generator/protobuf/generator.go; all 3 TestRPC*/TestNotification* tests pass GREEN.

## Must-Haves

- [ ] "A YANG module with an rpc statement emits a Protobuf service block named '<ModuleName>Service' containing an rpc method with matching Input and Output message references"
- [ ] "The Input and Output messages for an rpc are emitted as standalone message blocks"
- [ ] "A YANG module with an action statement emits the same service block structure as an rpc"
- [ ] "A YANG module with a notification statement emits a standalone message named '<NotificationName>Notification' — it does NOT appear as an rpc method in the service block"
- [ ] "TestRPCServiceBlock, TestActionServiceBlock, TestNotificationMessage all pass GREEN"

## Files

- `generator/protobuf/generator.go`
