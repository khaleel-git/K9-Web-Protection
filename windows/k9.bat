@echo off
echo Service started at %date% %time% >> C:\Logs\k9-service.log
:loop
tasklist /FI "IMAGENAME eq K9 Web Protection.exe" 2>NUL | find /I /N "K9 Web Protection.exe">NUL
if "%ERRORLEVEL%"=="1" (
  echo %date% %time% - K9 is down! Restarting... >> C:\Logs\k9-service.log
  start /MIN "" "C:\Program Files\K9 Web Protection\K9 Web Protection.exe"
  timeout /t 10 >nul
) else (
  timeout /t 5 >nul
)
goto loop