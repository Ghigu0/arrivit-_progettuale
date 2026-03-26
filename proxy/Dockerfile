FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o proxy

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/proxy /proxy

USER nonroot:nonroot
ENTRYPOINT ["/proxy"]