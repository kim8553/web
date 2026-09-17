@echo off
setlocal
chcp 65001 >nul 2>nul

echo Nine Yin Stage60 scene-bootstrap crash isolation
echo.
echo Gate 0 = PlayerEntry only; suppress every sendPlayerSpawn frame
echo Gate 1 = allow first existing sendPlayerSpawn frame
echo Gate 2 = allow first two frames
echo Gate 3 = allow first three frames ^(recommended first test^)
echo Gate 4 = allow first four frames
echo Gate 5 = allow all five sendPlayerSpawn frames, but withhold later post-spawn output
echo.
set "JIUYIN_STAGE60_BOOTSTRAP_GATE="
set /p "JIUYIN_STAGE60_BOOTSTRAP_GATE=Bootstrap gate [0-5, default 3]: "
if not defined JIUYIN_STAGE60_BOOTSTRAP_GATE set "JIUYIN_STAGE60_BOOTSTRAP_GATE=3"
echo STAGE60_BOOTSTRAP_GATE=%JIUYIN_STAGE60_BOOTSTRAP_GATE%
echo.
call "%~dp0RUN_STAGE60_LIVE.bat"
endlocal
