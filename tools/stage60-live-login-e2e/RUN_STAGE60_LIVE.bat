@echo off
setlocal EnableExtensions
title Nine Yin Stage60 LIVE Login E2E

set "ROOT=%~1"
if "%ROOT%"=="" (
  echo Enter the extracted 9yin-go-server1 root path.
  echo Example: E:\9yin-go-server1\9yin-go-server1
  set /p "ROOT=ServerRoot: "
)
if "%ROOT%"=="" (
  echo ERROR: ServerRoot is required.
  pause
  exit /b 1
)

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0run-stage60-live-services.ps1" -ServerRoot "%ROOT%"
set "RC=%ERRORLEVEL%"
echo.
echo Stage60 launcher finished with exit code %RC%.
pause
exit /b %RC%
