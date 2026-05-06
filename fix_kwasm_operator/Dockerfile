FROM alpine:latest
# Installiamo solo grep che serve allo script
RUN apk add --no-cache grep

# Copiamo il binario estratto e lo script nella root dell'immagine
COPY containerd-shim-wasmedge-v1 /containerd-shim-wasmedge-v1
COPY install-shim.sh /install-shim.sh

RUN chmod +x /install-shim.sh

CMD ["/bin/sh", "/install-shim.sh"]