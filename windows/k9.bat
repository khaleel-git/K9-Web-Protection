@echo off

:: Check if the Logs directory exists; if not, create it
if not exist "C:\Logs" (
    mkdir "C:\Logs"
)

echo Service started at %date% %time% >> C:\Logs\k9-service.log

:loop
tasklist /FI "IMAGENAME eq K9 Web Protection.exe" 2>NUL | find /I /N "K9 Web Protection.exe">NUL

if "%ERRORLEVEL%"=="1" (
  echo %date% %time% - K9 is down! Restarting... >> C:\Logs\k9-service.log
  
  :: Using /MIN to ensure it stays out of view if manually triggered
  start /MIN "" "C:\Program Files\K9 Web Protection\K9 Web Protection.exe"
  
  :: Wait for the process to fully initialize before checking again
  timeout /t 10 >nul
) else (
  :: Check every 5 seconds if the process is still healthy
  timeout /t 5 >nul
)

goto loop