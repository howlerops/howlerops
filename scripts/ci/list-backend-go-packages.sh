#!/usr/bin/env bash
set -euo pipefail

TAGS="${GO_TAGS:-duckdb}"

go list -tags "${TAGS}" \
  ./cmd/server \
  ./internal/... \
  ./pkg/... \
  ./services
