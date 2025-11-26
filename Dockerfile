FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/soap-proxy ./cmd/soap-proxy

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /
COPY --from=build /bin/soap-proxy /soap-proxy

VOLUME ["/data"]
EXPOSE 8080 8081

USER nobody:nogroup

ENTRYPOINT ["/soap-proxy"]
