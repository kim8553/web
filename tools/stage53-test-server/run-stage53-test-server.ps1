# Stage53 test server launcher. Uses an existing server root; never overwrites the user's current EXE.
[CmdletBinding()]
param(
    [string]$ServerRoot = 'D:\9yin_server',
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ServerArgs
)
$ErrorActionPreference = 'Stop'
$exe = Join-Path $PSScriptRoot 'stage53-test-server.exe'
if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
    Write-Host 'BLOCKED: stage53-test-server.exe is missing from this package.'
    exit 2
}
if (-not (Test-Path -LiteralPath $ServerRoot -PathType Container)) {
    Write-Host "BLOCKED: server root not found: $ServerRoot"
    exit 2
}
if (-not (Test-Path -LiteralPath (Join-Path $ServerRoot 'resources') -PathType Container)) {
    Write-Host 'BLOCKED: resources folder was not found under the server root.'
    exit 2
}

$previousDSN = [Environment]::GetEnvironmentVariable('NINEYIN_MYSQL_DSN', 'Process')
$dsn = $previousDSN
try {
    if ([string]::IsNullOrWhiteSpace($dsn)) {
        $candidates = @(
            (Join-Path $ServerRoot 'mysql.env'),
            (Join-Path (Join-Path $ServerRoot 'server') 'mysql.env')
        )
        foreach ($path in $candidates) {
            if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { continue }
            foreach ($line in (Get-Content -LiteralPath $path -ErrorAction Stop)) {
                $text = $line.Trim()
                if ($text -match '^(?i:set\s+)?"?NINEYIN_MYSQL_DSN\s*=\s*(.+?)\s*"?\s*$') {
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
        Write-Host 'BLOCKED: NINEYIN_MYSQL_DSN was not found. Stage50 purchase path intentionally requires unified MySQL storage.'
        Write-Host 'Start MySQL and keep your existing mysql.env in the server root; credentials are not printed.'
        exit 2
    }
    [Environment]::SetEnvironmentVariable('NINEYIN_MYSQL_DSN', $dsn, 'Process')

    Write-Host 'Starting Stage53 test server from the supplied package.'
    Write-Host "Server root: $ServerRoot"
    Write-Host 'MySQL DSN: configured (value hidden)'
    Write-Host 'Original server executable is not modified.'
    Push-Location $ServerRoot
    try {
        & $exe @ServerArgs
        exit $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
}
finally {
    [Environment]::SetEnvironmentVariable('NINEYIN_MYSQL_DSN', $previousDSN, 'Process')
}
