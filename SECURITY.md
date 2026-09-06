# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in go-guardian, please report it privately:

1. **DO NOT** open a public GitHub issue
2. Email: [your-email@example.com] (replace with actual email)
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

We will respond within 48 hours and provide a timeline for fixes.

## Security Best Practices

When using go-guardian:

### Rate Limiting
- Use Redis-based distributed rate limiting in production
- Set appropriate QPS limits based on your infrastructure capacity
- Monitor `guardian_ratelimit_rejects_total` metric

### Circuit Breaker
- Configure failure thresholds based on SLA requirements
- Set timeout values based on P99 latency
- Monitor `guardian_circuit_breaker_state` metric

### Tracing
- Never log sensitive data in trace contexts
- Sanitize user input before adding to trace metadata
- Use secure transport for trace data

### Metrics
- Restrict `/metrics` endpoint access (use firewall rules or authentication)
- Never expose PII in metric labels
- Monitor for metric cardinality explosion

### Redis (for distributed rate limiting)
- Use Redis with authentication enabled
- Enable TLS for Redis connections in production
- Restrict Redis network access to application servers only

## Dependencies

go-guardian minimizes dependencies:
- Core packages have zero external dependencies
- Optional dependencies (Redis, Prometheus, gRPC) are well-maintained

We monitor all dependencies for security vulnerabilities using Dependabot.
