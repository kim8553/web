# Capture only the fixed, read-only SHOP_WIRE_OBSERVE header from a running server log.
# This script NEVER starts, stops, patches, or mutates the server/client/database.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('baseline', 'purchase')]
    [string]$Phase,
    [Parameter(Mandatory = $true)]
    [string]$LogPath,
    [string]$OutputDirectory = '.\shop_wire_evidence',
    [ValidateRange(3, 120)]
    [int]$Seconds = 15
)
$ErrorActionPreference = 'Stop'
$maxAppendedBytes = 8 * 1024 * 1024
if (-not (Test-Path -LiteralPath $LogPath -PathType Leaf)) {
    throw "Server log does not exist: $LogPath"
}
$absoluteLog = [System.IO.Path]::GetFullPath($LogPath)
$start = (Get-Item -LiteralPath $absoluteLog).Length
Write-Host "Capturing $Phase for $Seconds seconds. Purchase phase: click ordinary NPC Buy once during this window."
Start-Sleep -Seconds $Seconds
$share = [System.IO.FileShare]::ReadWrite -bor [System.IO.FileShare]::Delete
$stream = [System.IO.File]::Open($absoluteLog, [System.IO.FileMode]::Open, [System.IO.FileAccess]::Read, $share)
try {
    $end = $stream.Length
    if ($end -lt $start) { throw 'Server log was truncated or rotated during capture; retry with one continuous log.' }
    $added = $end - $start
    if ($added -gt $maxAppendedBytes) { throw 'Captured server log is larger than 8 MiB; use a shorter capture window.' }
    [void]$stream.Seek($start, [System.IO.SeekOrigin]::Begin)
    $data = New-Object byte[] ([int]$added)
    $offset = 0
    while ($offset -lt $data.Length) {
        $read = $stream.Read($data, $offset, $data.Length - $offset)
        if ($read -eq 0) { throw 'Server log ended unexpectedly during capture.' }
        $offset += $read
    }
}
finally {
    $stream.Dispose()
}
$text = [System.Text.Encoding]::UTF8.GetString($data)
$header = [regex]'SHOP_WIRE_OBSERVE opcode=0x[0-9A-Fa-f]{2} selector=-?\d+ value_count=\d+ types=\[[0-9,]*\]'
$filtered = New-Object 'System.Collections.Generic.List[string]'
foreach ($line in ($text -split '\r?\n')) {
    $match = $header.Match($line)
    if ($match.Success) { $filtered.Add($match.Value) }
}
[void](New-Item -ItemType Directory -Force -Path $OutputDirectory)
$output = Join-Path $OutputDirectory "$Phase.log"
# The fixed header excludes remote IP/port, login data, raw packets, arbitrary strings,
# and even the optional decoded shop ID. Only opcode/selector/count/type remain.
[System.IO.File]::WriteAllLines([System.IO.Path]::GetFullPath($output), $filtered.ToArray(), ([System.Text.UTF8Encoding]::new($false)))
Write-Host "Captured $($filtered.Count) sanitized observations -> $output"
if ($Phase -eq 'purchase' -and $filtered.Count -eq 0) {
    throw 'No SHOP_WIRE_OBSERVE records in purchase window; verify Stage22 diagnostic EXE, its server log path, and the game connection.'
}
