param(
    [Parameter(Mandatory=$true)]
    [string]$ServerRoot
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path $ServerRoot).Path
$exe = Join-Path $PSScriptRoot 'stage57-test-server.exe'

if (-not (Test-Path $exe)) { throw "Missing stage57-test-server.exe: $exe" }
if (-not (Test-Path (Join-Path $root 'go.mod'))) { throw "ServerRoot must be the extracted 9yin-go-server1 root containing go.mod: $root" }
$skill = Join-Path $root 'resources\modern\share\skill\skill_new.ini'
if (-not (Test-Path $skill)) { throw "Missing required original 9yin-go-server1 resource: $skill" }

$env:NINEYIN_SERVER_ROOT = $root
Write-Host "Starting Stage57 server from the current 9yin-go-server1 source lineage."
Write-Host "Runtime root: $root"
if ([string]::IsNullOrWhiteSpace($env:NINEYIN_MYSQL_DSN)) {
    Write-Host 'Storage: native JSON mode (no MySQL DSN configured).'
} else {
    Write-Host 'Storage: MySQL mode (NINEYIN_MYSQL_DSN is configured).'
}
Write-Host 'Missing optional stringname.idres and missing unproven drop_table.json no longer block startup.'
Write-Host 'Gift-box/drop-table behavior stays disabled when drop_table.json is absent; no fabricated drop data is used.'
& $exe
exit $LASTEXITCODE
