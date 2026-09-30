param(
    [ValidateSet("stop", "precompact", "session-end")]
    [string]$Event = "stop"
)

$ErrorActionPreference = "Stop"
try {
    $raw = [Console]::In.ReadToEnd()
    $cwd = ""
    $transcript = ""
    if ($raw) {
        $payload = $raw | ConvertFrom-Json
        if ($payload.cwd) { $cwd = [string]$payload.cwd }
        if ($payload.transcript_path) { $transcript = [string]$payload.transcript_path }
    }
    if (-not $cwd) { $cwd = (Get-Location).Path }
    $root = Split-Path -Parent $PSScriptRoot
    $exe = Join-Path $root "bin\mcp-context-server.exe"
    if (-not (Test-Path -LiteralPath $exe)) {
        exit 0
    }
    $args = @("checkpoint", "--path", $cwd, "--event", $Event, "--client", "claude")
    if ($transcript) { $args += @("--transcript", $transcript) }
    & $exe @args | Out-Null
} catch {
    Write-Error $_
}
exit 0
