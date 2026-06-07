# Called by the NSIS uninstaller before removing files.
# Cleans up all system integration: tasks, firewall, proxy, hosts, CA.

$InstallDir  = 'C:\Program Files\K10 Web Protection'
$ExeDst      = "$InstallDir\K10WebProtection.exe"
$WatchdogDst = "$InstallDir\k10_watchdog.ps1"

$ErrorActionPreference = 'SilentlyContinue'

# Stop watchdog and main process
Stop-ScheduledTask -TaskName 'K10WebProtection-Watchdog'
Get-Process -Name 'K10WebProtection' | Stop-Process -Force
Start-Sleep -Milliseconds 800

# Remove scheduled tasks
Unregister-ScheduledTask -TaskName 'K10WebProtection'          -Confirm:$false
Unregister-ScheduledTask -TaskName 'K10WebProtection-Watchdog' -Confirm:$false

# Remove Windows Firewall rule
netsh advfirewall firewall delete rule name='K10 Block QUIC' 2>$null | Out-Null

# Clear system proxy
$key = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
Set-ItemProperty -Path $key -Name ProxyEnable -Value 0

# Notify WinINet
try {
    Add-Type -TypeDefinition @'
using System;using System.Runtime.InteropServices;
public class WI3{[DllImport("wininet.dll")]public static extern bool InternetSetOption(IntPtr h,int o,IntPtr b,int l);}
'@ -ErrorAction Stop
    [WI3]::InternetSetOption([IntPtr]::Zero,39,[IntPtr]::Zero,0)|Out-Null
    [WI3]::InternetSetOption([IntPtr]::Zero,37,[IntPtr]::Zero,0)|Out-Null
} catch {}

# Remove K10 hosts entries
$hostsPath = "$env:SystemRoot\System32\drivers\etc\hosts"
if (Test-Path $hostsPath) {
    $c = [System.IO.File]::ReadAllText($hostsPath)
    $c = [System.Text.RegularExpressions.Regex]::Replace(
        $c, '(?s)\r?\n# K10-Web-Protection START.*?# K10-Web-Protection END\r?\n?', '')
    $c = [System.Text.RegularExpressions.Regex]::Replace(
        $c, '(?s)\r?\n# K10-SafeSearch START.*?# K10-SafeSearch END\r?\n?', '')
    [System.IO.File]::WriteAllText($hostsPath, $c)
    ipconfig /flushdns | Out-Null
}

# Remove K10 CA from trust stores
Get-ChildItem Cert:\LocalMachine\Root |
    Where-Object { $_.Subject -like '*K10 Web Protection*' } |
    Remove-Item
Get-ChildItem Cert:\CurrentUser\Root |
    Where-Object { $_.Subject -like '*K10 Web Protection*' } |
    Remove-Item

# Unlock files before NSIS removes them
foreach ($f in @($ExeDst, $WatchdogDst)) {
    if (Test-Path $f) { icacls $f /remove:d Everyone 2>$null | Out-Null }
}
