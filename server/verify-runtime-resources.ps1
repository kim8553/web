# Read-only preflight for cmd/protocol-probe/skill_catalog.go, main.go and zz_recovered_overlay.go.
# This script does not create directories, change game data, or contact MySQL.
param(
    [string]$Root = $PSScriptRoot,
    [switch]$RequireMySQL
)

$ErrorActionPreference = 'Stop'
$rootPath = [System.IO.Path]::GetFullPath($Root)
$missing = New-Object System.Collections.Generic.List[string]

# Source: skill_catalog.go loadSkillResourceTables / mustLoadCombatSkillCatalog.
# These are mandatory at package initialization, before main() or either listener.
$requiredFiles = @(
    'resources\modern\share\skill\skill_new.ini',
    'resources\modern\share\skill\skill_static.ini',
    'resources\modern\share\skill\skill_normal_varprop.ini',
    'resources\modern\share\skill\skill_lock_varprop.ini',
    'resources\modern\share\skill\skill_consume.ini',
    'resources\modern\share\skill\damage_calculate.ini',
    'resources\modern\share\skill\attack_hitshape.ini',
    'resources\modern\share\skill\attack_targetshape.ini',
    'resources\modern\share\skill\buff_new.ini',
    'resources\modern\share\skill\buff_static.ini',
    'resources\modern\share\skill\buff_varprop.ini',
    'resources\modern\ini\action\zhaoshi_player.ini',
    'resources\modern\ini\action\zhaoshi_player_2.ini',
    'resources\modern\ini\action\zhaoshi_player_dodge.ini',
    'resources\modern\ini\action\zhaoshi_player_parry.ini',
    'resources\modern\ini\action\zhaoshi_clone.ini',
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

# Source: npc_catalog.go loadNPCTemplateTables and scene_catalog_registry.go.
# Empty directories previously passed the preflight but cannot supply NPCs.
$npcTemplatesRelative = 'resources\modern\share\npc\npcconfig'
$npcTemplatesDir = Join-Path $rootPath $npcTemplatesRelative
if ([System.IO.Directory]::Exists($npcTemplatesDir)) {
    $npcTable = Get-ChildItem -LiteralPath $npcTemplatesDir -File -Filter '*.txt' -Recurse | Select-Object -First 1
    if ($null -eq $npcTable) {
        $missing.Add("$npcTemplatesRelative (no nested TXT template files)")
    }
}
$npcCreatorsRelative = 'resources\modern\share\creator\npc_creator'
$npcCreatorsDir = Join-Path $rootPath $npcCreatorsRelative
if ([System.IO.Directory]::Exists($npcCreatorsDir)) {
    $npcCreator = Get-ChildItem -LiteralPath $npcCreatorsDir -File -Filter '*.xml' -Recurse | Select-Object -First 1
    if ($null -eq $npcCreator) {
        $missing.Add("$npcCreatorsRelative (no XML creator files)")
    }
}
$weaponRelative = 'resources\modern\share\ini\effect\playerweapon'
$weaponDir = Join-Path $rootPath $weaponRelative
$iniFiles = @()
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
