@echo off
:loop
tasklist /FI "IMAGENAME eq K9 Web Protection.exe" 2>NUL | find /I /N "K9 Web Protection.exe">NUL
if "%ERRORLEVEL%"=="1" (
  echo K9 is down! Restarting...
  start "" "C:\Program Files\K9 Web Protection\K9 Web Protection.exe"
)
timeout /t 5 >nul
goto loop