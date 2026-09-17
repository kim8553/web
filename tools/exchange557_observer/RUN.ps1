[CmdletBinding()]
param(
    [int]$Pid = 0,
    [string]$OutDir = '.\exchange557_capture'
)
$ErrorActionPreference = 'Stop'
$argsList = @((Join-Path $PSScriptRoot 'run_observer.py'), '--out-dir', $OutDir)
if ($Pid -gt 0) { $argsList += @('--pid', "$Pid") }
python @argsList
if ($LASTEXITCODE -ne 0) { throw "observer failed with exit code $LASTEXITCODE" }
