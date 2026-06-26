# Security

Echo MCP is intentionally designed for local development, local AI-assisted
testing, and controlled CI/test environments.

Do not expose Echo MCP directly to the public internet or to untrusted networks.
The MVP does not include authentication or authorization.

## Security Boundaries

- The MCP control plane can configure simulator behavior.
- The REST data plane returns configured responses to the Application Under
  Test.
- Webhook-style event delivery can send outbound HTTP requests to configured
  application webhook endpoints.
- Webhook endpoint addresses must be configured by the developer or test
  harness.
- AI agents select configured endpoint names; they must not provide arbitrary
  outbound URLs.

If exposed incorrectly, Echo MCP could be abused as an outbound request relay or
DDoS helper.

## Recommended Use

- Bind or expose Echo MCP to localhost by default where possible.
- Use firewalling, container/network isolation, or private CI networking if
  running outside a local machine.
- Only configure webhook endpoints owned by the application/test environment.
- Do not configure webhook endpoints pointing to arbitrary third-party systems.
- Do not treat Echo MCP as a production service, API gateway, reverse proxy, or
  public webhook relay.

## Reporting Security Issues

Do not file public issues for suspected vulnerabilities.

Use GitHub private vulnerability reporting if it is enabled for the repository.
If it is not available, contact the maintainer through:

```text
https://nagorn.io
```

## Supported Versions

Before `v1.0.0`, only the latest published release is expected to receive
security fixes.
