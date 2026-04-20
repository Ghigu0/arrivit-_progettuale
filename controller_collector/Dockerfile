# build stage
FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o controller

# runtime stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/controller .

CMD ["./controller"]