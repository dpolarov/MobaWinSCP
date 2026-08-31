# MobaWinSCP

Open the current MobaXterm SSH location in WinSCP.

MobaWinSCP is a small Windows/Go launcher intended to bridge a live MobaXterm SSH terminal with WinSCP. It reads the live SSH environment, matches the connection to a saved MobaXterm SSH bookmark, reuses the bookmark's username/port/private key, and opens WinSCP in the terminal's current remote directory.

> Early development version. MobaXterm's bookmark serialization (`#109#0%...`) is an internal format and may change between versions.

## What is supported

- Detect host and SSH port from `SSH_CONNECTION`
- Detect remote user from `USER` / `LOGNAME`
- Detect current remote directory from `PWD`
- Parse SSH (`#109#0`) bookmarks from `MobaXterm.ini`
- Match bookmark by host + port + username
- Detect `.ppk`, `.pem`, `id_rsa`, and `id_ed25519` key paths without relying on one fixed bookmark-field index
- Resolve MobaXterm `_CurrentDrive_:` portable key paths
- Auto-detect common MobaXterm.ini locations
- Auto-detect WinSCP.exe
- `-dry-run` diagnostic mode

MobaWinSCP intentionally does **not** extract, decrypt, or pass stored passwords.

## Build

```powershell
go test ./...
go build -o MobaWinSCP.exe ./cmd/mobawinscp
```

GitHub Actions also builds a Windows executable on every push to `main`.

## First test

Copy `MobaWinSCP.exe` somewhere reachable from the MobaXterm terminal and, while connected through SSH, run:

```bash
MobaWinSCP.exe -dry-run
```

Expected output resembles:

```text
Session : adp\win1/vpn_1 -> root@65.21.182.155:2255
Remote  : /var/www/project
INI     : C:\...\MobaXterm.ini
Key     : C:\ADP\NEW_PRV.ppk
WinSCP  : C:\Program Files (x86)\WinSCP\WinSCP.exe sftp://root@65.21.182.155:2255/var/www/project/ /privatekey=C:\ADP\NEW_PRV.ppk
```

If auto-detection cannot find the files:

```bash
MobaWinSCP.exe -dry-run -ini 'C:\path\MobaXterm.ini' -winscp 'C:\Program Files (x86)\WinSCP\WinSCP.exe'
```

After dry-run looks correct, run without `-dry-run`.

## Testing without an SSH connection

Environment overrides are provided for diagnostics:

```powershell
$env:MOBWINSCP_HOST='65.21.182.155'
$env:MOBWINSCP_PORT='2255'
$env:MOBWINSCP_USER='root'
$env:MOBWINSCP_PWD='/var/www/project'
.\MobaWinSCP.exe -dry-run
```

## Planned

- MobaXterm macro/hotkey integration
- More MobaXterm bookmark variants
- SSH gateway / jump-host translation to WinSCP
- Better hostname/IP matching when `SSH_CONNECTION` exposes an address but the bookmark stores a DNS name
- Release workflow and versioned binaries

## Security

The launcher never needs the MobaXterm master password and deliberately ignores password-bearing bookmark fields. Authentication should use the bookmark's SSH private key, an SSH agent compatible with WinSCP, or WinSCP's own credential handling.
