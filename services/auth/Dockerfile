FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/auth ./services/auth

RUN go build -o /bin/auth ./services/auth/cmd/auth

FROM alpine:3.19

RUN adduser -D -g '' appuser
USER appuser

COPY --from=builder /bin/auth /bin/auth

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/bin/auth"]
