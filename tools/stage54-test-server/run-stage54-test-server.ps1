# Stage54 test launcher for the current source lineage descended from 9yin-go-server1.rar.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ServerRoot,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ServerArgs
)
$ErrorActionPreference = 'Stop'
$exe = Join-Path $PSScriptRoot 'stage54-test-server.exe'
if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
    Write-Host 'BLOCKED: stage54-test-server.exe is missing.'
    exit 2
}
$ServerRoot = [IO.Path]::GetFullPath($ServerRoot)
if (-not (Test-Path -LiteralPath $ServerRoot -PathType Container)) {
    Write-Host 'BLOCKED: supplied 9yin-go-server1 runtime root does not exist.'
    exit 2
}
if (-not (Test-Path -LiteralPath (Join-Path $ServerRoot 'go.mod') -PathType Leaf)) {
    Write-Host 'BLOCKED: go.mod was not found. Point -ServerRoot at the extracted/current 9yin-go-server1 root.'
    exit 2
}
$skillNew = Join-Path $ServerRoot 'resources\modern\share\skill\skill_new.ini'
if (-not (Test-Path -LiteralPath $skillNew -PathType Leaf)) {
    Write-Host 'BLOCKED: current runtime resource resources\modern\share\skill\skill_new.ini is missing.'
    Write-Host 'Do not substitute an older server resources folder.'
    exit 2
}
$previousRoot = [Environment]::GetEnvironmentVariable('NINEYIN_SERVER_ROOT', 'Process')
try {
    [Environment]::SetEnvironmentVariable('NINEYIN_SERVER_ROOT', $ServerRoot, 'Process')
    Write-Host 'Starting test server from the current 9yin-go-server1 source lineage.'
    Write-Host "Runtime root: $ServerRoot"
    if ([string]::IsNullOrWhiteSpace($env:NINEYIN_MYSQL_DSN)) {
        Write-Host 'Storage: native JSON mode (no MySQL DSN configured).'
    } else {
        Write-Host 'Storage: MySQL mode (DSN value hidden).'
    }
    Write-Host 'No historical D:\9yin_server runtime is assumed.'
    Push-Location $ServerRoot
    try {
        & $exe @ServerArgs
        $code = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
    exit $code
}
finally {
    [Environment]::SetEnvironmentVariable('NINEYIN_SERVER_ROOT', $previousRoot, 'Process')
}
