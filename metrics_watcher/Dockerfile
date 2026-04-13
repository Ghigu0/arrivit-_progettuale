# build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY metrics_watcher.go .

RUN go build -o agent metrics_watcher.go

# runtime stage
FROM alpine:3.19

WORKDIR /app

# --from=builder significa copia dallo stage chiamato builder (che è quello precedente )
COPY --from=builder /app/agent . 

EXPOSE 8080

CMD ["./agent"]