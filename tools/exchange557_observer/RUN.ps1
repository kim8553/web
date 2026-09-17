[CmdletBinding()]
param(
    # $PID is a read-only PowerShell automatic variable; do not name a parameter $Pid.
    [int]$ProcessId = 0,
    [string]$OutDir = '.\exchange557_capture'
)
$ErrorActionPreference = 'Stop'
$captureDirectory = Join-Path $OutDir ('run_{0}_{1}' -f (Get-Date -Format 'yyyyMMdd_HHmmss'), [guid]::NewGuid().ToString('N').Substring(0, 8))
$argsList = @((Join-Path $PSScriptRoot 'run_observer.py'), '--out-dir', $captureDirectory)
if ($ProcessId -gt 0) { $argsList += @('--pid', "$ProcessId") }
python @argsList
if ($LASTEXITCODE -ne 0) { throw "observer failed with exit code $LASTEXITCODE" }

# Each run gets a new directory: stale captures from earlier runs cannot produce a false PASS.
$captures = @(Get-ChildItem -LiteralPath $captureDirectory -File -Filter 'exchange557_*.txt')
if ($captures.Count -ne 1) { throw "expected one capture file; got $($captures.Count)" }
$configLines = @(Get-Content -LiteralPath $captures[0].FullName | Where-Object { $_.StartsWith('JIUYIN_EXCHANGE557_CONFIG=') })
if ($configLines.Count -eq 0) { throw 'No exchange 557 config captured. A current local server that rejects BindStatus=1 cannot provide the reference response.' }

$probe = Join-Path (Split-Path $PSScriptRoot -Parent) 'stage37_exchange557_bind_probe.py'
$parsed = Join-Path $captureDirectory 'parsed_557.json'
python $probe --input $captures[0].FullName --output $parsed
if ($LASTEXITCODE -ne 0) { throw "exchange557 parser rejected the capture (exit $LASTEXITCODE)" }
$result = Get-Content -LiteralPath $parsed -Raw -Encoding UTF8 | ConvertFrom-Json
$boundCases = @($result.records | Where-Object { $_.BindStatus -gt 0 })
if ($boundCases.Count -eq 0) { throw 'No BindStatus>0 reference response observed; the blocked 388 rows remain unverified.' }
Write-Host "PASS: $($result.record_count) parsed exchange configs; $($boundCases.Count) binding-sensitive configs."
Write-Host "Share only the .txt/parsed_557.json after reviewing them; JSONL may contain local file paths. Capture: $captureDirectory"
