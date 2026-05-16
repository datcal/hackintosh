# Build the Go host app for Windows.
# Usage:  .\scripts\build-host.ps1
$ErrorActionPreference = "Stop"

$root = Resolve-Path "$PSScriptRoot\.."
$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null

Push-Location (Join-Path $root "host")
try {
    $version = (git -C $root rev-parse --short HEAD 2>$null)
    if (-not $version) { $version = "dev" }
    $env:CGO_ENABLED = "0"
    $out = Join-Path $dist "hackintosh.exe"
    go build `
        -ldflags "-X main.version=$version -s -w" `
        -o $out `
        ./cmd/hackintosh
    Write-Host "Built $out (version $version)"
} finally {
    Pop-Location
}
