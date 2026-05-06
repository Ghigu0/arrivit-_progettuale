FROM scratch

COPY target/wasm32-wasip1/release/controller_wasm.wasm /controller_wasm.wasm

ENTRYPOINT ["/controller_wasm.wasm"]
