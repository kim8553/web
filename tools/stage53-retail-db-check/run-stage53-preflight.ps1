# Stage53 local-only DB schema inspection. Never uploads credentials or database data.
# Extract alongside retail-schema-preflight.exe. Run only on a trusted PC.
[CmdletBinding()]
param([string]$ServerRoot = '')
$ErrorActionPreference = 'Stop'
$probe = Join-Path $PSScriptRoot 'retail-schema-preflight.exe'
$reportPath = Join-Path $PSScriptRoot 'stage53_report.json'
if (-not (Test-Path -LiteralPath $probe -PathType Leaf)) {
    Write-Host 'BLOCKED: retail-schema-preflight.exe is missing from this package.'
    exit 2
}
if (-not $ServerRoot) { $ServerRoot = $PSScriptRoot }
if (-not (Test-Path -LiteralPath $ServerRoot -PathType Container)) {
    Write-Host 'BLOCKED: the specified server folder does not exist.'
    exit 2
}
$previousDSN = [Environment]::GetEnvironmentVariable('NINEYIN_MYSQL_DSN', 'Process')
$dsn = $previousDSN
try {
    if ([string]::IsNullOrWhiteSpace($dsn)) {
        # Only these local locations are inspected; never search or upload a user's drive.
        $candidates = @(
            (Join-Path $ServerRoot 'mysql.env'),
            (Join-Path (Join-Path $ServerRoot 'server') 'mysql.env')
        )
        foreach ($path in $candidates) {
            if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { continue }
            foreach ($line in (Get-Content -LiteralPath $path -ErrorAction Stop)) {
                if ($line -match '^\s*(?:set\s+)?"?NINEYIN_MYSQL_DSN\s*=\s*(.+?)\s*"?\s*$') {
                    $candidate = $Matches[1].Trim()
                    if ($candidate.Length -ge 2 -and $candidate.StartsWith('"') -and $candidate.EndsWith('"')) {
                        $candidate = $candidate.Substring(1, $candidate.Length - 2)
                    }
                    if ($candidate.Length -gt 0) { $dsn = $candidate; break }
                }
            }
            if (-not [string]::IsNullOrWhiteSpace($dsn)) { break }
        }
    }
    if ([string]::IsNullOrWhiteSpace($dsn)) {
        Write-Host 'BLOCKED: no local NINEYIN_MYSQL_DSN was found. No database was accessed.'
        exit 2
    }
    # The DSN is passed exclusively through the child process environment, never CLI or output.
    [Environment]::SetEnvironmentVariable('NINEYIN_MYSQL_DSN', $dsn, 'Process')
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $probe
    $psi.WorkingDirectory = $PSScriptRoot
    $psi.UseShellExecute = $false
    $psi.CreateNoWindow = $true
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $psi
    if (-not $process.Start()) { throw 'preflight did not start' }
    $stdout = $process.StandardOutput.ReadToEnd()
    $stderr = $process.StandardError.ReadToEnd()
    if (-not $process.WaitForExit(45000)) {
        $process.Kill()
        Write-Host 'BLOCKED: database inspection timed out. No changes requested.'
        exit 2
    }
    if ([string]::IsNullOrWhiteSpace($stdout)) {
        # The executable intentionally sanitizes its own errors; do not print raw diagnostics.
        Write-Host 'BLOCKED: inspection could not produce a report. Check DB availability locally.'
        exit 2
    }
    try { $parsed = $stdout | ConvertFrom-Json -ErrorAction Stop }
    catch {
        Write-Host 'BLOCKED: inspection returned an invalid report.'
        exit 2
    }
    if ($null -eq $parsed.status -or $null -eq $parsed.findings) {
        Write-Host 'BLOCKED: inspection report format is incomplete.'
        exit 2
    }
    $utf8 = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($reportPath, ($stdout.TrimEnd() + "`n"), $utf8)
    if ($process.ExitCode -ne 0 -or $parsed.status -ne 'PASS') {
        Write-Host 'BLOCKED: read-only DB inspection completed with findings; stage53_report.json saved locally.'
        exit 2
    }
    Write-Host 'PASS: read-only DB schema preflight passed. stage53_report.json saved locally.'
    Write-Host 'This is NOT a game purchase, server boot, or reconnect test.'
}
catch {
    # Never print exception details: those might contain local paths or credentials.
    Write-Host 'BLOCKED: local inspection failed. No database repair was attempted.'
    exit 2
}
finally {
    [Environment]::SetEnvironmentVariable('NINEYIN_MYSQL_DSN', $previousDSN, 'Process')
}
