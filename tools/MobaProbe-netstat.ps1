# Optional read-only helper for manual diagnostics if WMI TCP enumeration is unavailable.
Get-NetTCPConnection | Sort-Object OwningProcess,LocalAddress,LocalPort | Format-Table -AutoSize
