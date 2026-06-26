# Terminology

## Core Terms

**AI Agent**

An automated coding or testing agent that may use the control plane to
configure Echo MCP during a test.

**Application Under Test**

The application being tested. It interacts with Echo MCP through normal
data-plane interfaces and should not need MCP awareness or simulator-specific
logic.

**Control Plane**

The MCP interface used by AI agents and other MCP clients to configure Echo MCP.
The control plane is for simulator control, not for normal application data
exchange.

**Data Plane**

The normal service interface through which the application under test and a
simulated service exchange data, such as HTTP, WebSocket, webhook, or similar
interfaces. The data plane must remain separate from MCP control.

**External Dependency**

A service outside the application boundary, such as a payment gateway, identity
provider, CRM, shipping provider, banking API, or third-party webhook provider.

**Simulated Service**

An Echo MCP representation of one external dependency that participates in the
data plane as if it were that dependency. In the current implementation, one
Echo MCP process represents one simulated service.

**Contract**

A description of the expected service surface for a simulated dependency.
Current implementation support is limited to developer-provided OpenAPI 3.0.x
JSON as a validation constraint for configured REST behavior. A contract is not
a source of generated behavior, built-in public API behavior, or automatic
scenario creation.

**Scenario**

A test-time configuration that groups behavior for a simulated service during a
test or test segment. The current MVP supports one active REST behavior rule per
Echo MCP process.

**Behavior Rule**

A match-and-outcome pair that tells Echo MCP what simulated behavior to apply
when a matching data-plane interaction occurs.

**Request Match**

The criteria used to decide whether a behavior rule applies to an incoming
data-plane request or interaction.

**Outcome**

The simulated result of a matched interaction. The current MVP supports
configured HTTP responses. Other outcome types, such as timeout, delay, retry
sequences, or signature-related failures, are not current capabilities unless a
later release explicitly adds them.

**Network Condition**

A simulated communication characteristic, such as latency, timeout, connection
interruption, or retry-relevant response timing. This term does not imply
general-purpose chaos engineering. Network-condition outcomes are not part of
the current MVP.

**Webhook Event**

A data-plane event delivered to an application endpoint through webhook-style
transport. It is defined from the application under test's perspective and does
not assume which system originates the event. Current webhook-style event
delivery is limited to one immediate HTTP `POST` to a registered application
webhook endpoint selected by endpoint name.

**Observation**

Information available for verifying data-plane interactions and simulated
outcomes, such as received requests, matched rules, emitted responses, timing
effects, or event delivery attempts.
