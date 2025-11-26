# SOAP mTLS Proxy with Trace UI

Simple HTTP proxy that:

- Accepts plain HTTP from a client/service
- Forwards to a downstream SOAP-like HTTPS endpoint using mTLS
- Captures requests/responses and stores them in a JSONL file
- Exposes a web UI to browse traces

## Quick start (local)

```bash
go mod tidy
make build
```

Or run directly:

```bash
UPSTREAM_URL=https://downstream.example.com/soap \
TRACE_FILE=./traces.jsonl \
MTLS_CERT_FILE=./certs/client.crt \
MTLS_KEY_FILE=./certs/client.key \
MTLS_CA_FILE=./certs/ca.crt \
go run ./cmd/soap-proxy
```

Proxy listens on :8080, UI on :8081.
