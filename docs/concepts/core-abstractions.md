# Core Abstractions

These abstractions describe the conceptual model for Echo MCP. They do not add
features beyond the implemented release surface.

## Simulated Service

Each simulated service represents a single external dependency from the
perspective of the application under test. The application under test interacts
with it through the data plane as if it were that dependency.

Examples include a payment gateway, identity provider, CRM, shipping provider,
banking API, or third-party webhook provider.

This abstraction does not imply that Echo MCP is a production provider or a
complete service-virtualization platform.

In the current implementation, one Echo MCP process represents one simulated
service. Multi-dependency support inside one process is future work and would
require a future ADR.

## Contract

A contract describes the expected surface area of a simulated service.

For the current REST validation slice, Echo MCP can load one
developer-provided OpenAPI 3.0.x JSON contract as a validation constraint for
configured behavior. Echo MCP validates behavior against the contract before
accepting it; it does not generate, infer, repair, or complete behavior from the
contract.

Broader contract formats, contract upload, OpenAPI generation, AsyncAPI, and
contract-driven scenario generation remain outside this abstraction unless a
later task or ADR explicitly adds them.

## Scenario

A scenario is a runtime configuration used for a test or test segment.

A scenario groups behavior rules so an AI agent can configure a dependency to
behave in a specific way for a specific test case. This abstraction does not
decide whether scenarios are stored, scripted, named, or represented as data.
The current MVP supports one active REST behavior rule per Echo MCP process.

## Behavior Rule

A behavior rule describes when Echo MCP should apply a simulated outcome to a
data-plane interaction.

A behavior rule has two conceptual parts:

- match criteria for a data-plane interaction
- outcome to apply when the criteria match

Behavior rules are configured through the control plane and exercised through
the data plane.

## Request Match

A request match identifies which data-plane interaction a behavior rule applies
to.

The current MVP matches REST requests by method and path. Matches may eventually
involve request body fields, headers, transaction identifiers, call order, or
other test-relevant criteria, but those are not current capabilities.

## Outcome

An outcome is the simulated result produced when a behavior rule matches a
data-plane interaction.

The current MVP supports configured HTTP responses, including success and error
status codes. Future outcome types could include:

- timeout
- delay
- retry-relevant response sequence
- signature-validation failure condition
- scoped rejection

## Network Condition

A network condition is a simulated communication characteristic applied to a
data-plane interaction.

Examples include latency, timeout, connection interruption, and retry-relevant
response timing. This abstraction supports targeted dependency simulation; it
does not imply general-purpose chaos engineering. Network-condition outcomes are
not part of the current MVP.

## Webhook Event

A webhook event is a data-plane event delivered to an application endpoint
through webhook-style transport. From the application under test's perspective,
it is inbound.

The current webhook slice sends one immediate HTTP `POST` with
`Content-Type: application/json` to a registered application webhook endpoint
selected by endpoint name. Endpoint addresses are configured by the developer or
test harness, not supplied as raw outbound destinations by the MCP client.

Retries, scheduling, signing, delivery persistence, AsyncAPI, event buses, and
inbound public webhook receiving remain outside the current abstraction.

## Observation

An observation is information available for verifying data-plane interactions
and simulated outcomes.

Observations may include received requests, selected behavior rules, produced
outcomes, timing effects, or event delivery attempts. This abstraction does not
decide storage, retrieval, retention, or evidence format.

## Conceptual Request/Response Flow

One common request/response flow is:

1. An AI agent configures simulated service behavior through the control plane.
2. The application under test sends a data-plane interaction to Echo MCP.
3. Echo MCP evaluates the active scenario and behavior rules.
4. Echo MCP produces the selected outcome through the data plane.
5. Observation information is available to support test verification.

The first MVP request/response slice intentionally targeted REST-style
request/response interactions. ADR-0005 later added a separate webhook-style
event delivery slice while preserving the same control-plane/data-plane
separation.
