#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# generate.sh  –  generate Go gRPC stubs from proto/manga.proto
#
# Prerequisites (run once):
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#
# Make sure $GOPATH/bin (or $HOME/go/bin) is on your PATH:
#   export PATH="$PATH:$(go env GOPATH)/bin"
#
# Then run this script from the Backend/ directory:
#   chmod +x proto/generate.sh
#   ./proto/generate.sh
# ─────────────────────────────────────────────────────────────────────────────

set -e

# Move to the Backend root regardless of where the script is called from
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "▶  Generating Go code from proto/manga.proto ..."

protoc \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  proto/manga.proto

echo "✅  Generated:"
echo "     proto/manga.pb.go"
echo "     proto/manga_grpc.pb.go"
