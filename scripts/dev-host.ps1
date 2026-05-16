# Run the host app without building. Use --probe to test the MCU link,
# or no flags to run the full app (once milestones 4+ land).
$ErrorActionPreference = "Stop"
$root = Resolve-Path "$PSScriptRoot\.."
Push-Location (Join-Path $root "host")
try {
    go run ./cmd/hackintosh @args
} finally {
    Pop-Location
}
