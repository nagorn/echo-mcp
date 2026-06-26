# Unmatched REST Requests

If the Application Under Test calls Echo MCP's REST data plane before any
matching behavior has been configured, Echo MCP returns:

```text
HTTP 501 Not Implemented
```

This indicates simulator setup failure. It is not a simulated external provider
response.

## Why 501?

The response is deterministic and fails loudly. It avoids ambiguity with
provider-like responses such as `404`, `500`, or `503`, which should be returned
only when explicitly configured through `configure_behavior`.

## What AI Agents Should Do

When a test receives HTTP `501` from Echo MCP:

1. Treat it as missing simulator setup.
2. Inspect available observations and application logs.
3. Configure the missing behavior through the MCP control plane.
4. Rerun the test.

Observation information for unmatched REST requests may be improved later, but
that is not required for the current MVP behavior.
