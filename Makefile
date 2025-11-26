APP_NAME := soap-proxy
IMAGE := your-registry/soap-proxy:latest

build:
	go build -o bin/$(APP_NAME) ./cmd/soap-proxy

docker-build:
	docker build -t $(IMAGE) .

docker-push:
	docker push $(IMAGE)

run-local:
	UPSTREAM_URL=https://example.com/soap \
	TRACE_FILE=./traces.jsonl \
	MTLS_CERT_FILE=./certs/client.crt \
	MTLS_KEY_FILE=./certs/client.key \
	MTLS_CA_FILE=./certs/ca.crt \
	go run ./cmd/soap-proxy
