# Called by the NSIS installer after copying files.
# Receives the install directory as the first argument.
param([string]$InstallDir = 'C:\Program Files\K10 Web Protection')

$ExePath      = "$InstallDir\K10WebProtection.exe"
$WatchdogPath = "$InstallDir\k10_watchdog.ps1"

# Task Scheduler — main app (AtLogOn for all users, restart every 12 s on exit)
$action    = New-ScheduledTaskAction -Execute $ExePath
$trigger   = New-ScheduledTaskTrigger -AtLogOn
$settings  = New-ScheduledTaskSettingsSet `
    -RestartCount 999 -RestartInterval (New-TimeSpan -Seconds 12) `
    -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew
$principal = New-ScheduledTaskPrincipal -GroupId 'BUILTIN\Users' -RunLevel Highest
Register-ScheduledTask -TaskName 'K10WebProtection' `
    -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null

# Task Scheduler — watchdog (SYSTEM, AtStartup, loops indefinitely)
$wAction    = New-ScheduledTaskAction -Execute 'powershell.exe' `
    -Argument "-NonInteractive -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$WatchdogPath`""
$wTrigger   = New-ScheduledTaskTrigger -AtStartup
$wSettings  = New-ScheduledTaskSettingsSet `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -RestartCount 999 -RestartInterval (New-TimeSpan -Seconds 30)
$wPrincipal = New-ScheduledTaskPrincipal -UserId 'SYSTEM' -RunLevel Highest -LogonType ServiceAccount
Register-ScheduledTask -TaskName 'K10WebProtection-Watchdog' `
    -Action $wAction -Trigger $wTrigger -Settings $wSettings -Principal $wPrincipal -Force | Out-Null

# Windows Firewall — block QUIC (UDP 443) so browsers cannot bypass the TCP proxy
netsh advfirewall firewall delete rule name='K10 Block QUIC' 2>$null | Out-Null
netsh advfirewall firewall add rule name='K10 Block QUIC' `
    dir=out protocol=UDP localport=443 action=block | Out-Null

# Lock files so they cannot be deleted without Administrator + explicit unlock
icacls $ExePath      /deny 'Everyone:(DE,WD,AD)' 2>$null | Out-Null
icacls $WatchdogPath /deny 'Everyone:(DE,WD,AD)' 2>$null | Out-Null
