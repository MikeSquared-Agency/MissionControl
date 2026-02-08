FROM golang:1.22-bookworm AS go-builder
WORKDIR /src
COPY orchestrator/go.mod orchestrator/go.sum ./orchestrator/
COPY cmd/mc/go.mod cmd/mc/go.sum ./cmd/mc/
RUN cd orchestrator && go mod download
RUN cd cmd/mc && go mod download
COPY . .
RUN cd cmd/mc && CGO_ENABLED=0 go build -o /usr/local/bin/mc .

FROM rust:1.75-bookworm AS rust-builder
WORKDIR /src
COPY core/ ./core/
RUN cd core && cargo build --release
RUN cp core/target/release/mc-core /usr/local/bin/mc-core 2>/dev/null || true

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=go-builder /usr/local/bin/mc /usr/local/bin/mc
COPY --from=rust-builder /usr/local/bin/mc-core /usr/local/bin/mc-core
EXPOSE 8080
WORKDIR /workspace
ENTRYPOINT ["mc", "serve", "--api-only"]
