FROM golang:1.22 AS builder
WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o proxy

FROM alpine:3.19
WORKDIR /

COPY --from=builder /app/proxy /proxy

RUN chmod +x /proxy

ENTRYPOINT ["/proxy"]