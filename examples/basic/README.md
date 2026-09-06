# Basic Example

Demonstrates all middleware components working together:

- **Rate Limiting**: 10 QPS with 20 burst capacity
- **Circuit Breaker**: Opens after 5 consecutive failures
- **Fallback Strategy**: Returns canned response on LLM failure
- **Distributed Tracing**: Auto-generated trace_id for each request
- **Prometheus Metrics**: Exposed at `/metrics`

## Run

```bash
cd examples/basic
go run main.go
```

## Test

```bash
# Normal request
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"query": "Hello AI"}'

# With custom trace ID
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "X-Trace-ID: my-custom-trace-123" \
  -d '{"query": "How are you?"}'

# Trigger rate limit (send 25 requests quickly)
for i in {1..25}; do
  curl -X POST http://localhost:8080/chat \
    -H "Content-Type: application/json" \
    -d "{\"query\": \"Request $i\"}" &
done
wait

# Health check
curl http://localhost:8080/health

# Prometheus metrics
curl http://localhost:8080/metrics | grep llm
```

## Expected Metrics

```promql
# Total requests
ai_mechanic_requests_total{agent_id="chat",status="success"}

# Request latency P95
histogram_quantile(0.95, rate(ai_mechanic_request_duration_seconds_bucket[5m]))

# LLM call duration
ai_mechanic_llm_call_duration_seconds{model="gpt-4",node="chat_node"}

# Rate limit rejections
ai_mechanic_ratelimit_rejects_total{limiter="ingress"}

# Circuit breaker state (0=closed, 1=open, 2=half_open)
ai_mechanic_circuit_breaker_state{service="llm"}
```
