#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/schemas/generated.schema.json"

echo "Generating $OUT from xcw schema"
cd "$ROOT"
go run ./cmd/xcw schema > "$OUT"
echo "Generated schema; commit to keep schemas in sync."
