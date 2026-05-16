#!/usr/bin/env bash
# Run the host app without building. Use --probe to test the MCU link,
# or no flags to run the full app (once milestones 4+ land).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root/host"
go run ./cmd/hackintosh "$@"
