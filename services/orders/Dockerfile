FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY services/orders ./services/orders

RUN go build -o /bin/orders ./services/orders/cmd/orders

FROM alpine:3.19

RUN adduser -D -g '' appuser
USER appuser

COPY --from=builder /bin/orders /bin/orders

ENV PORT=8081
EXPOSE 8081

ENTRYPOINT ["/bin/orders"]
