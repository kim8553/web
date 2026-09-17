param(
    [Parameter(Mandatory=$true)]
    [string]$ServerRoot
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path $ServerRoot).Path
$exe = Join-Path $PSScriptRoot 'stage59-live-server.exe'
$lister = Join-Path $PSScriptRoot 'loopback-lister.ps1'
$logs = Join-Path $PSScriptRoot 'logs'
$listerLogs = Join-Path $logs 'lister'
$runtimeLog = Join-Path $root 'artifacts\local\logs\protocol-probe-live.log'

if (-not (Test-Path $exe)) { throw "Missing stage59-live-server.exe: $exe" }
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

function Quote-ChildArg([string]$Value) {
    if ($Value.Contains('"')) { throw "Child-process path contains an unsupported quote character: $Value" }
    return '"' + $Value + '"'
}

$gameOwners = @(Get-PortOwners 19061)
$gmOwners = @(Get-PortOwners 19062)
if ($gameOwners.Count -gt 0 -or $gmOwners.Count -gt 0) {
    throw "Port 19061 or 19062 is already in use. Close the previous Nine Yin test server first. No process was killed."
}

$env:NINEYIN_SERVER_ROOT = $root
Write-Host 'Starting Stage59 full local services from the 9yin-go-server1 source lineage.'
Write-Host "Runtime root : $root"
Write-Host 'Server list  : 127.0.0.1:4000'
Write-Host 'Game         : 127.0.0.1:19061'
Write-Host 'GM UI        : http://127.0.0.1:19062/'
if ([string]::IsNullOrWhiteSpace($env:NINEYIN_MYSQL_DSN)) {
    Write-Host 'Storage      : native JSON mode'
    $backupDir = Join-Path $root 'data\stage59-backups'
    New-Item -ItemType Directory -Force -Path $backupDir | Out-Null
    $stamp = Get-Date -Format 'yyyyMMdd_HHmmss_fff'
    $backup = Join-Path $backupDir ("roles.before-stage59-login.$stamp.json")
    Copy-Item -LiteralPath $roles -Destination $backup -ErrorAction Stop
    $sourceHash = (Get-FileHash -LiteralPath $roles -Algorithm SHA256).Hash.ToLower()
    $backupHash = (Get-FileHash -LiteralPath $backup -Algorithm SHA256).Hash.ToLower()
    if ($sourceHash -ne $backupHash) { throw 'Stage59 roles.json backup hash mismatch; refusing to start.' }
    Write-Host "Stage59 JSON backup: $backup"
    Write-Host "Backup SHA256       : $backupHash"
    Write-Host 'First successful Stage59 login may rebind one eligible legacy acct:v1 identity in data\roles.json.'
} else {
    Write-Host 'Storage      : MySQL mode (DSN value is not printed)'
    Write-Host 'Stage59 JSON identity adoption is inactive in MySQL mode.'
}

$listerOwners = @(Get-PortOwners 4000)
if ($listerOwners.Count -eq 0) {
    $listerArgs = @('-NoProfile','-ExecutionPolicy','Bypass','-File',(Quote-ChildArg $lister),'-Port','4000','-OutputDir',(Quote-ChildArg $listerLogs))
    $listerProc = Start-Process -FilePath 'powershell.exe' -ArgumentList $listerArgs -WorkingDirectory $PSScriptRoot -PassThru
    Write-Host "Started server-list service PID=$($listerProc.Id)"
} else {
    Write-Host "Server-list port 4000 is already listening; reusing existing listener PID(s): $($listerOwners -join ',')"
}

$serverArgs = @('-listen','127.0.0.1:19061','-gm-listen','127.0.0.1:19062')
$serverProc = Start-Process -FilePath $exe -ArgumentList $serverArgs -WorkingDirectory $root -PassThru
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
Write-Host '=== Stage59 listener verification ==='
Write-Host "4000  server-list : $($p4000.Count -gt 0)"
Write-Host "19061 game        : $($p19061.Count -gt 0)"
Write-Host "19062 GM          : $($p19062.Count -gt 0)"

if ($serverProc.HasExited -or $p4000.Count -eq 0 -or $p19061.Count -eq 0 -or $p19062.Count -eq 0) {
    Write-Host ''
    Write-Host 'STAGE59_READY=NO'
    Write-Host "Runtime log: $runtimeLog"
    if (Test-Path $runtimeLog) {
        Write-Host '--- last runtime log lines ---'
        Get-Content $runtimeLog -Tail 30
    }
    throw 'Stage59 local service stack did not become ready. Do not launch the game client yet.'
}

Write-Host ''
Write-Host 'STAGE59_READY=YES'
Write-Host 'All three original local-service ports are listening. Keep this window and the server-list/game windows open.'
Write-Host 'Next LIVE gate: login must reach ServerPlayerRoles(opcode=0x04); listener readiness alone is not login success.'
Write-Host "Runtime log: $runtimeLog"
Write-Host ''
Write-Host 'Press Enter only when you want this launcher window to close. It does not kill the server processes.'
[void](Read-Host)
