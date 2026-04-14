FROM golang:1.22 AS builder
WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o collector metrics_collector.go

FROM alpine:3.19
WORKDIR /
COPY --from=builder /app/collector /collector

ENTRYPOINT ["/collector"]