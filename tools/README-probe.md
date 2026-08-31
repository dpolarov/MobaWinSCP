# MobaProbe

Read-only diagnostic helper for investigating MobaXterm runtime state without sending input to SSH sessions.

Usage:

```powershell
.\MobaProbe.exe snapshot A
# switch active SSH tab in MobaXterm manually
.\MobaProbe.exe snapshot B
.\MobaProbe.exe diff .\mobaprobe-A.txt .\mobaprobe-B.txt
```

The probe collects process metadata, windows, window properties, TCP connections, and Moba/SSH/SFTP-named entries in the current user's temp directory. It does not send keystrokes, terminal commands, clipboard data, or messages to MobaXterm.
