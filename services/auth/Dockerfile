FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY proto ./proto
COPY services/auth ./services/auth

RUN go build -o /bin/auth ./services/auth/cmd/auth

# Download grpc_health_probe for health checks
RUN GRPC_HEALTH_PROBE_VERSION=v0.4.25 && \
    wget -qO/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /bin/grpc_health_probe

FROM alpine:3.19

RUN adduser -D -g '' appuser

COPY --from=builder /bin/auth /bin/auth
COPY --from=builder /bin/grpc_health_probe /bin/grpc_health_probe

USER appuser

ENV GRPC_PORT=9090
EXPOSE 9090

ENTRYPOINT ["/bin/auth"]
