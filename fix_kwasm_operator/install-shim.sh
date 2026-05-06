#!/bin/sh
set -e

echo "--- FIX DEFINITIVO K3S V3 ---"

BIN_DEST="/host/opt/kwasm/bin"
# Usiamo la cartella degli extra per non farci sovrascrivere da K3s
EXTRA_CONFIG_DIR="/host/var/lib/rancher/k3s/agent/etc/containerd/config-v3.toml.d"
EXTRA_CONFIG_FILE="$EXTRA_CONFIG_DIR/wasmedge.toml"

# 1. Copia Binario (già verificato, ma lo rifacciamo per sicurezza)
mkdir -p "$BIN_DEST"
cp /containerd-shim-wasmedge-v1 "$BIN_DEST/"
chmod +x "$BIN_DEST/containerd-shim-wasmedge-v1"
echo "[OK] Binario presente in /opt/kwasm/bin"

# 2. Scrittura file extra (Sintassi corretta per containerd v3)
mkdir -p "$EXTRA_CONFIG_DIR"
cat <<EOF > "$EXTRA_CONFIG_FILE"
[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.wasmedge]
  runtime_type = "io.containerd.wasmedge.v1"
[plugins."io.containerd.cri.v1.runtime".containerd.runtimes.wasmedge.options]
  BinaryName = "/opt/kwasm/bin/containerd-shim-wasmedge-v1"
EOF
echo "[OK] Creato file di configurazione extra: $EXTRA_CONFIG_FILE"

# 3. Riavvio "Brutale" (visto che nsenter systemctl dava noie)
echo "3. Riavvio runtime..."
# Uccidiamo containerd, K3s lo riavvierà subito e leggerà i nuovi file .toml
nsenter --target 1 --mount --uts --ipc --net --pid pkill -f containerd || echo "Containerd non trovato o già riavviato"

echo "--- PROCEDURA ULTIMATA ---"
sleep infinity