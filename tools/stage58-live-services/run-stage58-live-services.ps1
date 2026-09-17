param(
    [Parameter(Mandatory=$true)]
    [string]$ServerRoot
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path $ServerRoot).Path
$exe = Join-Path $PSScriptRoot 'stage58-live-server.exe'
$lister = Join-Path $PSScriptRoot 'loopback-lister.ps1'
$logs = Join-Path $PSScriptRoot 'logs'
$listerLogs = Join-Path $logs 'lister'
$gameLog = Join-Path $logs 'game-19061.current.log'
$runtimeLog = Join-Path $logs 'protocol-probe-live.log'

if (-not (Test-Path $exe)) { throw "Missing stage58-live-server.exe: $exe" }
if (-not (Test-Path $lister)) { throw "Missing loopback-lister.ps1: $lister" }
if (-not (Test-Path (Join-Path $root 'go.mod'))) { throw "ServerRoot must be the extracted 9yin-go-server1 root containing go.mod: $root" }
$skill = Join-Path $root 'resources\modern\share\skill\skill_new.ini'
if (-not (Test-Path $skill)) { throw "Missing required original 9yin-go-server1 resource: $skill" }
$roles = Join-Path $root 'data\roles.json'
if (-not (Test-Path $roles)) { throw "Missing original 9yin-go-server1 role data: $roles" }

New-Item -ItemType Directory -Force -Path $logs | Out-Null
New-Item -ItemType Directory -Force -Path $listerLogs | Out-Null

function Get-PortOwners([int]$Port) {
    return @(Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -Unique)
}

$gameOwners = @(Get-PortOwners 19061)
$gmOwners = @(Get-PortOwners 19062)
if ($gameOwners.Count -gt 0 -or $gmOwners.Count -gt 0) {
    throw "Port 19061 or 19062 is already in use. Close the previous Nine Yin test server first. No process was killed."
}

$env:NINEYIN_SERVER_ROOT = $root
Write-Host 'Starting Stage58 full local services from the 9yin-go-server1 source lineage.'
Write-Host "Runtime root : $root"
Write-Host 'Server list  : 127.0.0.1:4000'
Write-Host 'Game         : 127.0.0.1:19061'
Write-Host 'GM UI        : http://127.0.0.1:19062/'
if ([string]::IsNullOrWhiteSpace($env:NINEYIN_MYSQL_DSN)) {
    Write-Host 'Storage      : native JSON mode'
} else {
    Write-Host 'Storage      : MySQL mode (DSN value is not printed)'
}

$listerOwners = @(Get-PortOwners 4000)
if ($listerOwners.Count -eq 0) {
    $listerArgs = @('-NoProfile','-ExecutionPolicy','Bypass','-File',$lister,'-Port','4000','-OutputDir',$listerLogs)
    $listerProc = Start-Process -FilePath 'powershell.exe' -ArgumentList $listerArgs -WorkingDirectory $PSScriptRoot -PassThru
    Write-Host "Started server-list service PID=$($listerProc.Id)"
} else {
    Write-Host "Server-list port 4000 is already listening; reusing existing listener PID(s): $($listerOwners -join ',')"
}

$serverArgs = @('-listen','127.0.0.1:19061','-gm-listen','127.0.0.1:19062','-log-file',$runtimeLog)
$serverProc = Start-Process -FilePath $exe -ArgumentList $serverArgs -WorkingDirectory $root -RedirectStandardOutput $gameLog -RedirectStandardError $gameLog -PassThru
Write-Host "Started game service PID=$($serverProc.Id)"

$deadline = (Get-Date).AddSeconds(20)
do {
    Start-Sleep -Milliseconds 500
    if ($serverProc.HasExited) { break }
    $p4000 = @(Get-PortOwners 4000)
    $p19061 = @(Get-PortOwners 19061)
    $p19062 = @(Get-PortOwners 19062)
    if ($p4000.Count -gt 0 -and $p19061.Count -gt 0 -and $p19062.Count -gt 0) { break }
} while ((Get-Date) -lt $deadline)

$p4000 = @(Get-PortOwners 4000)
$p19061 = @(Get-PortOwners 19061)
$p19062 = @(Get-PortOwners 19062)

Write-Host ''
Write-Host '=== Stage58 listener verification ==='
Write-Host "4000  server-list : $($p4000.Count -gt 0)"
Write-Host "19061 game        : $($p19061.Count -gt 0)"
Write-Host "19062 GM          : $($p19062.Count -gt 0)"

if ($serverProc.HasExited -or $p4000.Count -eq 0 -or $p19061.Count -eq 0 -or $p19062.Count -eq 0) {
    Write-Host ''
    Write-Host 'STAGE58_READY=NO'
    Write-Host "Game log: $gameLog"
    if (Test-Path $gameLog) {
        Write-Host '--- last game log lines ---'
        Get-Content $gameLog -Tail 30
    }
    throw 'Stage58 local service stack did not become ready. Do not launch the game client yet.'
}

Write-Host ''
Write-Host 'STAGE58_READY=YES'
Write-Host 'All three original local-service ports are listening. Keep this window and the server-list window open.'
Write-Host 'This proves local listeners only; latest-client LIVE/E2E is still unverified until an actual client connects.'
Write-Host "Game log: $gameLog"
Write-Host ''
Write-Host 'Press Enter only when you want this launcher window to close. It does not kill the server processes.'
[void](Read-Host)
