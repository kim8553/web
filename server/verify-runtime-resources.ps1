# Read-only preflight for cmd/protocol-probe/main.go and zz_recovered_overlay.go.
# This script does not create directories, change game data, or contact MySQL.
param(
    [string]$Root = $PSScriptRoot,
    [switch]$RequireMySQL
)

$ErrorActionPreference = 'Stop'
$rootPath = [System.IO.Path]::GetFullPath($Root)
$missing = New-Object System.Collections.Generic.List[string]

# All entries below come from current Go loader paths, not inferred client paths.
$requiredFiles = @(
    'resources\modern\share\skill\qinggong\qgdefine.ini',
    'resources\modern\share\item\tool_item.ini',
    'resources\modern\share\item\equipment.ini',
    'resources\modern\share\item\itemartstatic.ini',
    'resources\modern\share\item\drop_table.json',
    'resources\modern\text\stringname.idres'
)
$requiredDirectories = @(
    'resources\modern\share\npc\npcconfig',
    'resources\modern\share\creator\npc_creator',
    'resources\modern\share\ini\effect\playerweapon'
)
foreach ($relative in $requiredFiles) {
    $absolute = Join-Path $rootPath $relative
    if (-not [System.IO.File]::Exists($absolute)) {
        $missing.Add($relative)
    } elseif ((Get-Item -LiteralPath $absolute).Length -eq 0) {
        $missing.Add("$relative (empty)")
    }
}
foreach ($relative in $requiredDirectories) {
    if (-not [System.IO.Directory]::Exists((Join-Path $rootPath $relative))) {
        $missing.Add($relative)
    }
}
$weaponRelative = 'resources\modern\share\ini\effect\playerweapon'
$weaponDir = Join-Path $rootPath $weaponRelative
if ([System.IO.Directory]::Exists($weaponDir)) {
    $iniFiles = @(Get-ChildItem -LiteralPath $weaponDir -File -Filter '*.ini')
    if ($iniFiles.Count -eq 0) {
        $missing.Add("$weaponRelative (no INI files)")
    }
    foreach ($file in $iniFiles) {
        if ($file.Length -eq 0) {
            $missing.Add("$weaponRelative\$($file.Name) (empty)")
        }
    }
}
if ($RequireMySQL -and [string]::IsNullOrWhiteSpace($env:NINEYIN_MYSQL_DSN)) {
    $missing.Add('NINEYIN_MYSQL_DSN (not set; refusing legacy JSON mode)')
}
if ($missing.Count -ne 0) {
    [Console]::Error.WriteLine("RESOURCE_PREFLIGHT=FAIL root=$rootPath")
    foreach ($entry in $missing) {
        [Console]::Error.WriteLine("MISSING: $entry")
    }
    [Console]::Error.WriteLine('Source files must match the current server loader. Do not make empty directories or fabricate INI/JSON data.')
    exit 1
}
Write-Host "RESOURCE_PREFLIGHT=PASS root=$rootPath weapon_ini=$($iniFiles.Count) mysql_required=$([bool]$RequireMySQL)"
exit 0
