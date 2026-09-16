@echo off
setlocal EnableExtensions
chcp 65001 >nul
for %%I in ("%~dp0..") do set "ROOT=%%~fI"
set "NINEYIN_SERVER_ROOT=%ROOT%"
set "SERVER=%~dp0stage33-shop-diagnostic.exe"
set "SHOP_PATH=%ROOT%\resources\modern\share\trade\shop.ini"
set "QG_PATH=%ROOT%\resources\modern\share\skill\qinggong\qgdefine.ini"
set "LISTER=%ROOT%\loopback-lister.ps1"
set "LOGDIR=%ROOT%\logs\stage33-shop-diagnostic"
set "NINEYIN_SHOP_WIRE_TRACE=1"
set "NINEYIN_SHOP_EXCHANGE_VIEW_AB=0"
set "EXPECTED_SHOP_SHA=ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd"
set "EXPECTED_EXE_SHA=6647b4101e5fe9b3358077be72479866dc07f7f12507c5bfb5181fe4093797e9"

if not exist "%SERVER%" goto :missing
if not exist "%SHOP_PATH%" goto :missing
if not exist "%QG_PATH%" goto :missing
if not exist "%LISTER%" goto :missing

powershell.exe -NoProfile -Command "$s=(Get-FileHash -Algorithm SHA256 -LiteralPath $env:SHOP_PATH).Hash; $e=(Get-FileHash -Algorithm SHA256 -LiteralPath $env:SERVER).Hash; if ($s -ne $env:EXPECTED_SHOP_SHA -or $e -ne $env:EXPECTED_EXE_SHA) { Write-Host 'STOP: expected RAR shop.ini or verified EXE checksum mismatch'; Write-Host ('shop.ini: '+$s); Write-Host ('server:   '+$e); exit 3 }; exit 0"
if errorlevel 1 goto :hash_failure

powershell.exe -NoProfile -Command "$p=@(Get-NetTCPConnection -State Listen -LocalPort 4000,19061,19062 -ErrorAction SilentlyContinue); if ($p.Count -gt 0) { $p | Select-Object LocalAddress,LocalPort,OwningProcess | Format-Table; exit 4 }; exit 0"
if errorlevel 1 goto :port_busy

if not exist "%LOGDIR%" mkdir "%LOGDIR%"
echo.
echo Stage33 diagnostic: original 9yin-game-native-menu.exe will NOT be changed.
echo NPC shop wire observation ON; exchange view experimental mode OFF.
echo GM web listener disabled for this diagnostic copy.
echo Logs: %LOGDIR%\stage33_game.log
echo.
start "Stage33 Local Server List" /D "%ROOT%" powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%LISTER%" -Port 4000 -OutputDir "%LOGDIR%\lister"
start "Stage33 Nine Yin Game" /D "%ROOT%" "%SERVER%" -listen 127.0.0.1:19061 -gm-listen= -log-file "%LOGDIR%\stage33_game.log"
echo Start requests issued; actual login and shop display remain unverified.
echo Review the server window and the log for startup errors.
pause
exit /b 0

:missing
echo STOP: kit must be inside the existing extracted 9yin-go-server1 root.
echo Required: kit EXE, resources\modern\share\trade\shop.ini,
echo resources\modern\share\skill\qinggong\qgdefine.ini, loopback-lister.ps1.
pause
exit /b 2
:hash_failure
echo STOP: this kit has NOT started the server due to a SHA256 mismatch.
echo Do not overwrite or substitute resources from another client version.
pause
exit /b 3
:port_busy
echo STOP: port 4000, 19061, or 19062 is in use.
echo This script does NOT kill the running server. Close it yourself if appropriate.
pause
exit /b 4
