#!/usr/bin/env bash
# test.sh — run all backend tests (unit + integration).
# Integration tests require the docker-compose stack to be running.
set -e
cd "$(dirname "$0")"

echo "==> Running unit tests"
go test ./... -count=1 -race

echo
echo "==> Running integration tests (requires docker-compose up)"
go test -tags=integration ./internal/... -count=1 -v

echo
echo "==> All backend tests passed"