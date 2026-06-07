# K10 Web Protection — Integrity Watchdog
# Runs as SYSTEM via Task Scheduler at system startup.
# Loops every 10 seconds: re-locks files, ensures the app is running,
# re-enables the proxy if it was cleared, and re-applies the QUIC firewall rule.

$ExePath      = 'C:\Program Files\K10 Web Protection\K10WebProtection.exe'
$WatchdogPath = 'C:\Program Files\K10 Web Protection\k10_watchdog.ps1'
$ProxyPort    = 8080
$TaskName     = 'K10WebProtection'
$FirewallRule = 'K10 Block QUIC'

function k10-running {
    $null -ne (Get-Process -Name 'K10WebProtection' -ErrorAction SilentlyContinue)
}

function proxy-enabled {
    $v = Get-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' `
             -Name 'ProxyEnable' -ErrorAction SilentlyContinue
    $v -and $v.ProxyEnable -eq 1
}

function notify-wininet {
    try {
        Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class WI{[DllImport("wininet.dll")]public static extern bool InternetSetOption(IntPtr h,int o,IntPtr b,int l);}
'@ -ErrorAction Stop
        [WI]::InternetSetOption([IntPtr]::Zero,39,[IntPtr]::Zero,0)|Out-Null
        [WI]::InternetSetOption([IntPtr]::Zero,37,[IntPtr]::Zero,0)|Out-Null
    } catch {}
}

function set-proxy([bool]$on) {
    $key = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
    if ($on) {
        Set-ItemProperty -Path $key -Name ProxyEnable -Value 1 -ErrorAction SilentlyContinue
        Set-ItemProperty -Path $key -Name ProxyServer  -Value "127.0.0.1:$ProxyPort" -ErrorAction SilentlyContinue
    } else {
        Set-ItemProperty -Path $key -Name ProxyEnable -Value 0 -ErrorAction SilentlyContinue
    }
    notify-wininet
}

function lock-file([string]$path) {
    if (Test-Path $path) {
        icacls $path /deny 'Everyone:(DE,WD,AD)' 2>$null | Out-Null
    }
}

function firewall-rule-present {
    (netsh advfirewall firewall show rule name="$FirewallRule" 2>$null) -match $FirewallRule
}

function ensure-task {
    if (-not (Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue)) {
        $action    = New-ScheduledTaskAction -Execute $ExePath
        $trigger   = New-ScheduledTaskTrigger -AtLogOn
        $settings  = New-ScheduledTaskSettingsSet -RestartCount 999 `
                         -RestartInterval (New-TimeSpan -Seconds 12) `
                         -ExecutionTimeLimit ([TimeSpan]::Zero) `
                         -MultipleInstances IgnoreNew
        $principal = New-ScheduledTaskPrincipal -GroupId 'BUILTIN\Users' -RunLevel Highest
        Register-ScheduledTask -TaskName $TaskName `
            -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Force | Out-Null
    }
}

while ($true) {
    # Re-lock files if permissions were stripped
    lock-file $ExePath
    lock-file $WatchdogPath

    # Ensure the Task Scheduler entry wasn't deleted
    ensure-task

    # Re-apply QUIC block if Windows Firewall rule was removed
    if (-not (firewall-rule-present)) {
        netsh advfirewall firewall add rule name="$FirewallRule" `
            dir=out protocol=UDP localport=443 action=block 2>$null | Out-Null
    }

    # If K10 died while the proxy was still enabled, clear the proxy so internet
    # traffic isn't broken, then restart the app.
    if (-not (k10-running)) {
        if (proxy-enabled) { set-proxy $false }
        if (Test-Path $ExePath) {
            Start-Process -FilePath $ExePath -WindowStyle Hidden -ErrorAction SilentlyContinue
        }
    }

    Start-Sleep -Seconds 10
}
