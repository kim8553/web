param(
    [Parameter(Mandatory=$true)]
    [string]$ServerRoot
)

$ErrorActionPreference = 'Stop'
$root = (Resolve-Path -LiteralPath $ServerRoot).Path
$exe = Join-Path $PSScriptRoot 'stage60-live-server.exe'
$lister = Join-Path $PSScriptRoot 'loopback-lister.ps1'
$roles = Join-Path $root 'data\roles.json'
$skill = Join-Path $root 'resources\modern\share\skill\skill_new.ini'
$runtimeLog = Join-Path $root 'artifacts\local\logs\protocol-probe-live.log'
$stamp = Get-Date -Format 'yyyyMMdd_HHmmss_fff'
$resultDir = Join-Path $PSScriptRoot ("stage60-results\$stamp")
$listerDir = Join-Path $resultDir 'lister'
$summary = Join-Path $resultDir 'STAGE60_SUMMARY.txt'
$resultZip = Join-Path $PSScriptRoot ("STAGE60_LIVE_RESULT_$stamp.zip")

foreach ($required in @($exe,$lister,(Join-Path $root 'go.mod'),$roles,$skill)) {
    if (-not (Test-Path -LiteralPath $required)) { throw "Stage60 required path missing: $required" }
}
New-Item -ItemType Directory -Force -Path $resultDir,$listerDir | Out-Null

function PortOwners([int]$Port) {
    @(Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty OwningProcess -Unique)
}
function Add-Summary([string]$Name,[string]$Value) {
    "$Name=$Value" | Add-Content -LiteralPath $summary -Encoding UTF8
}
function Capture-Ports([string]$Name) {
    $rows = foreach ($port in 4000,19061,19062) {
        Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue |
            Select-Object LocalAddress,LocalPort,RemoteAddress,RemotePort,State,OwningProcess
    }
    @($rows) | Format-Table -AutoSize | Out-String |
        Set-Content -LiteralPath (Join-Path $resultDir $Name) -Encoding UTF8
}
function Capture-Log([string]$Name,[int]$SkipLines) {
    $target = Join-Path $resultDir $Name
    if (-not (Test-Path -LiteralPath $runtimeLog)) {
        '' | Set-Content -LiteralPath $target -Encoding UTF8
        return
    }
    $all = @(Get-Content -LiteralPath $runtimeLog -ErrorAction SilentlyContinue)
    if ($all.Count -gt $SkipLines) {
        @($all | Select-Object -Skip $SkipLines) | Set-Content -LiteralPath $target -Encoding UTF8
    } else {
        '' | Set-Content -LiteralPath $target -Encoding UTF8
    }
}
function Stop-Own($Proc,[string]$Label) {
    if ($null -ne $Proc -and -not $Proc.HasExited) {
        Stop-Process -Id $Proc.Id -Force -ErrorAction SilentlyContinue
        Add-Summary ($Label + '_STOPPED') 'YES'
    }
}

$busy = @()
foreach ($port in 4000,19061,19062) {
    $owners = @(PortOwners $port)
    if ($owners.Count -gt 0) { $busy += "$port:$($owners -join ',')" }
}
if ($busy.Count -gt 0) {
    throw "Stage60 refuses to reuse old listeners. Close previous Nine Yin test processes first. Busy: $($busy -join '; ')"
}

$preLogLines = 0
if (Test-Path -LiteralPath $runtimeLog) {
    $preLogLines = @(Get-Content -LiteralPath $runtimeLog -ErrorAction SilentlyContinue).Count
}
$rolesBefore = (Get-FileHash -LiteralPath $roles -Algorithm SHA256).Hash.ToLower()
'' | Set-Content -LiteralPath $summary -Encoding UTF8
Add-Summary 'STAGE' '60'
Add-Summary 'SERVER_LINEAGE' '9yin-go-server1.rar -> kim8553/web'
Add-Summary 'RUNTIME_ROOT' $root
Add-Summary 'SERVER_EXE_SHA256' ((Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash.ToLower())
Add-Summary 'ROLES_SHA256_BEFORE' $rolesBefore
Add-Summary 'START_UTC' ((Get-Date).ToUniversalTime().ToString('o'))

$env:NINEYIN_SERVER_ROOT = $root
if ([string]::IsNullOrWhiteSpace($env:NINEYIN_MYSQL_DSN)) {
    Add-Summary 'STORAGE_MODE' 'JSON'
    $backupDir = Join-Path $root 'data\stage60-backups'
    New-Item -ItemType Directory -Force -Path $backupDir | Out-Null
    $backup = Join-Path $backupDir ("roles.before-stage60-login.$stamp.json")
    Copy-Item -LiteralPath $roles -Destination $backup -ErrorAction Stop
    $backupHash = (Get-FileHash -LiteralPath $backup -Algorithm SHA256).Hash.ToLower()
    Add-Summary 'ROLES_BACKUP' $backup
    Add-Summary 'ROLES_BACKUP_SHA256' $backupHash
    if ($backupHash -ne $rolesBefore) { throw 'Stage60 roles.json backup SHA256 mismatch; refusing to start.' }
} else {
    Add-Summary 'STORAGE_MODE' 'MYSQL'
    Add-Summary 'MYSQL_DSN_PRINTED' 'NO'
}

$listerProc = $null
$serverProc = $null
$hadError = $false
try {
    $listerArgs = @('-NoProfile','-ExecutionPolicy','Bypass','-File',('"'+$lister+'"'),'-Port','4000','-OutputDir',('"'+$listerDir+'"'))
    $listerProc = Start-Process -FilePath 'powershell.exe' -ArgumentList $listerArgs -WorkingDirectory $PSScriptRoot -PassThru
    $serverProc = Start-Process -FilePath $exe -ArgumentList @('-listen','127.0.0.1:19061','-gm-listen','127.0.0.1:19062') `
        -WorkingDirectory $root -PassThru `
        -RedirectStandardOutput (Join-Path $resultDir 'server.stdout.log') `
        -RedirectStandardError (Join-Path $resultDir 'server.stderr.log')

    $deadline = (Get-Date).AddSeconds(90)
    $ready = $false
    do {
        Start-Sleep -Milliseconds 500
        if ($serverProc.HasExited -or $listerProc.HasExited) { break }
        $ready = (@(PortOwners 4000).Count -gt 0 -and @(PortOwners 19061).Count -gt 0 -and @(PortOwners 19062).Count -gt 0)
    } while (-not $ready -and (Get-Date) -lt $deadline)
    Capture-Ports 'ports.ready.txt'
    Add-Summary 'LOCAL_SERVICES_READY' ($(if ($ready) {'YES'} else {'NO'}))
    if (-not $ready) { throw 'Stage60 local service stack did not become ready. Do not launch the client.' }

    Write-Host ''
    Write-Host 'STAGE60_READY=YES'
    Write-Host 'Run the CURRENT Snail client. Login with the existing account, select the existing character, and enter the scene if possible.'
    Write-Host 'When the first attempt reaches the scene OR stops/fails, return here and press Enter.'
    [void](Read-Host)

    Capture-Log 'protocol-probe.after-first-attempt.txt' $preLogLines
    Capture-Ports 'ports.after-first-attempt.txt'
    Add-Summary 'ROLES_SHA256_AFTER_FIRST_ATTEMPT' ((Get-FileHash -LiteralPath $roles -Algorithm SHA256).Hash.ToLower())
    $firstPath = Join-Path $resultDir 'protocol-probe.after-first-attempt.txt'
    $firstLog = Get-Content -Raw -LiteralPath $firstPath -ErrorAction SilentlyContinue
    Add-Summary 'LIST_REQUEST_CAPTURED' ($(if (Test-Path -LiteralPath (Join-Path $listerDir 'request.bin')) {'YES'} else {'NO'}))
    Add-Summary 'LOGIN_OPCODE_0X02_SEEN' ($(if ($firstLog -match 'opcode=0x02 decoded_len=') {'YES'} else {'NO'}))
    Add-Summary 'ROLE_LIST_SENT' ($(if ($firstLog -match 'sent ServerPlayerRoles\(opcode=0x04, role_exists=true\)') {'YES'} else {'NO'}))
    Add-Summary 'ROLE_CHOSEN_SCENE_INIT' ($(if ($firstLog -match 'waiting for ClientReady') {'YES'} else {'NO'}))
    Add-Summary 'CLIENT_READY_ACTIVITY_SEEN' ($(if ($firstLog -match 'target ClientReady|ignored duplicate ClientReady in active scene') {'YES'} else {'NO'}))

    Write-Host ''
    Write-Host 'First-attempt evidence captured.'
    Write-Host 'If it FAILED, do not retry; press Enter now.'
    Write-Host 'If it REACHED THE SCENE, disconnect normally, reconnect once with the same account/character, enter the scene again, then press Enter.'
    [void](Read-Host)

    Capture-Log 'protocol-probe.after-reconnect-attempt.txt' $preLogLines
    Capture-Ports 'ports.after-reconnect-attempt.txt'
    Add-Summary 'ROLES_SHA256_AFTER_RECONNECT_ATTEMPT' ((Get-FileHash -LiteralPath $roles -Algorithm SHA256).Hash.ToLower())
    $fullPath = Join-Path $resultDir 'protocol-probe.after-reconnect-attempt.txt'
    $fullLog = Get-Content -Raw -LiteralPath $fullPath -ErrorAction SilentlyContinue
    Add-Summary 'ROLE_LIST_SENT_COUNT' ([string][regex]::Matches($fullLog,'sent ServerPlayerRoles\(opcode=0x04, role_exists=true\)').Count)
    Add-Summary 'SCENE_INIT_COUNT' ([string][regex]::Matches($fullLog,'waiting for ClientReady').Count)
    Add-Summary 'END_UTC' ((Get-Date).ToUniversalTime().ToString('o'))
}
catch {
    $hadError = $true
    Add-Summary 'LAUNCHER_ERROR' ($_.Exception.Message.Replace("`r",' ').Replace("`n",' '))
    Capture-Log 'protocol-probe.stage60-error.txt' $preLogLines
    Write-Host "STAGE60_ERROR=$($_.Exception.Message)"
}
finally {
    Stop-Own $serverProc 'SERVER'
    Stop-Own $listerProc 'LISTER'
    if (Test-Path -LiteralPath $roles) {
        Add-Summary 'ROLES_SHA256_FINAL' ((Get-FileHash -LiteralPath $roles -Algorithm SHA256).Hash.ToLower())
    }
    Get-ChildItem -LiteralPath $resultDir -File -Recurse | ForEach-Object {
        $relative = $_.FullName.Substring($resultDir.Length).TrimStart('\')
        $h = Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256
        "$($h.Hash.ToLower())  $relative"
    } | Set-Content -LiteralPath (Join-Path $resultDir 'SHA256SUMS.txt') -Encoding ascii
    if (Test-Path -LiteralPath $resultZip) { Remove-Item -LiteralPath $resultZip -Force }
    Compress-Archive -Path (Join-Path $resultDir '*') -DestinationPath $resultZip -CompressionLevel Optimal
    Write-Host ''
    Write-Host "STAGE60_RESULT_ZIP=$resultZip"
    Write-Host "STAGE60_RESULT_SHA256=$((Get-FileHash -LiteralPath $resultZip -Algorithm SHA256).Hash.ToLower())"
    Write-Host 'Send this one STAGE60_LIVE_RESULT_*.zip back for failure-point classification.'
}
if ($hadError) { exit 1 }
