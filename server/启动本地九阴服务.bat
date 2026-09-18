@echo off
setlocal EnableExtensions DisableDelayedExpansion
title Nine Yin Portable Server

REM Keep launcher, EXE and resources under the same ROOT; do not hardcode drive letters.
set "ROOT=%~dp0"
set "NINEYIN_SERVER_ROOT=%ROOT%"
set "SERVER=%ROOT%9yin-game-native-menu.exe"
set "LOGDIR=%ROOT%logs"
set "LISTER=%ROOT%loopback-lister.ps1"
set "PREFLIGHT=%ROOT%verify-runtime-resources.ps1"

if not exist "%SERVER%" (
  echo ERROR: server executable not found: "%SERVER%"
  pause
  exit /b 1
)
if not exist "%LISTER%" (
  echo ERROR: lister script not found: "%LISTER%"
  pause
  exit /b 1
)
if not exist "%PREFLIGHT%" (
  echo ERROR: resource preflight script not found: "%PREFLIGHT%"
  pause
  exit /b 1
)

REM Load existing local CMD-format mysql.env without printing or packaging credentials.
if exist "%ROOT%mysql.env" for /f "usebackq delims=" %%L in ("%ROOT%mysql.env") do call %%L
if not defined NINEYIN_MYSQL_DSN (
  echo ERROR: NINEYIN_MYSQL_DSN not set. Refusing to silently use JSON storage.
  echo Keep your existing mysql.env next to this launcher; do not publish its contents.
  pause
  exit /b 1
)

REM Read-only checks first, so a missing playerweapon folder is reported before DB gates.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%PREFLIGHT%" -Root "%ROOT%" -RequireMySQL
if errorlevel 1 (
  echo ERROR: required server resources are missing or invalid. Nothing was started.
  pause
  exit /b 1
)

REM The current Go runner only verifies existing migration checksums when
REM NINEYIN_ALLOW_SCHEMA_MIGRATIONS is not exactly YES. Never authorize DDL
REM from this launcher, including when mysql.env or the parent shell sets YES.
if /I "%NINEYIN_ALLOW_SCHEMA_MIGRATIONS%"=="YES" (
  echo ERROR: migration authorization YES detected; this safe launcher will not execute schema changes.
  echo Review your existing mysql.env separately. It has not been changed.
  pause
  exit /b 1
)
set "NINEYIN_ALLOW_SCHEMA_MIGRATIONS=NO"
REM Go VerifyApplied will STOP without changing the DB if the migration ledger
REM is missing or mismatched. This is not a full schema compatibility audit.
if not exist "%LOGDIR%" mkdir "%LOGDIR%"

powershell.exe -NoProfile -Command "$listeners = @(Get-NetTCPConnection -State Listen -LocalPort 19061,19062 -ErrorAction SilentlyContinue); if ($listeners.Count -eq 0) { exit 0 }; $owners = @($listeners.OwningProcess | Sort-Object -Unique); if ($owners.Count -eq 1) { $process = Get-CimInstance Win32_Process -Filter ('ProcessId=' + $owners[0]); if ($process.ExecutablePath -and [IO.Path]::GetFullPath($process.ExecutablePath) -eq [IO.Path]::GetFullPath($env:SERVER)) { exit 2 } }; exit 1"
set "PORT_STATE=%ERRORLEVEL%"
if "%PORT_STATE%"=="2" (
  echo Nine Yin server copy is already running.
  start "Nine Yin GM Panel" http://127.0.0.1:19062/
  exit /b 0
)
if "%PORT_STATE%"=="1" (
  echo ERROR: game or GM port is already in use.
  echo Close the other Nine Yin game server before starting this copy.
  pause
  exit /b 1
)

echo.
echo Starting portable Nine Yin local services...
echo Server list : 127.0.0.1:4000
echo Game  : 127.0.0.1:19061
echo GM UI : http://127.0.0.1:19062/
echo Storage: MySQL ^(credentials hidden^)
echo.

powershell.exe -NoProfile -Command "if (Get-NetTCPConnection -State Listen -LocalPort 4000 -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }"
if errorlevel 1 (
  start "Nine Yin Server List 4000" /D "%ROOT%" powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%LISTER%" -Port 4000
) else (
  echo Server list 4000 is already running; reusing it.
)
start "Nine Yin Game 19061" /D "%ROOT%" "%SERVER%" -listen 127.0.0.1:19061 -gm-listen 127.0.0.1:19062 -log-file "%LOGDIR%\protocol-probe-live.log" 1>>"%LOGDIR%\game-19061.current.log" 2>&1
timeout /t 2 /nobreak >nul
start "Nine Yin GM Panel" http://127.0.0.1:19062/
echo Launcher finished. Check server logs before interpreting GM/game availability.
pause
