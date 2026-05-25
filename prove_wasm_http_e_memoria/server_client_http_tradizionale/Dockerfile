FROM debian:bookworm-slim

WORKDIR /app

COPY target/release/linux_http_client /app/linux_http_client

RUN chmod +x /app/linux_http_client

EXPOSE 8080

CMD ["/app/linux_http_client"]