# TunGo Installation Guide

Quick installation scripts for TunGo Client on all platforms (macOS, Linux, Windows).

## Features

✅ **No Admin Required** - Installs to user-writable directory  
✅ **Auto-Detection** - Detects OS and architecture automatically  
✅ **Latest Version** - Always downloads the latest release  
✅ **Cross-Platform** - macOS, Linux, and Windows support  

## Installation

### Linux & macOS

Run the installation script:

```bash
curl -sSL https://raw.githubusercontent.com/sombochea/tungo/main/scripts/install.sh | bash
```

Or clone and run locally:

```bash
./scripts/install.sh
```

**Installation Location:** `~/.local/bin/tungo`

### Windows

Run the PowerShell installation script (no admin required):

```powershell
powershell -ExecutionPolicy Bypass -Command "Invoke-WebRequest -Uri 'https://raw.githubusercontent.com/sombochea/tungo/main/scripts/install.ps1' -OutFile install.ps1; .\install.ps1"
```

Or run locally:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

**Installation Location:** `%APPDATA%\TunGo\bin\tungo.exe`

### Custom Installation Directory

#### Linux & macOS

```bash
TUNGO_INSTALL_DIR="/path/to/custom/dir" ./scripts/install.sh
```

#### Windows

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -InstallDir "C:\CustomPath\TunGo"
```

## After Installation

### Add to PATH

The script will prompt if the installation directory needs to be added to PATH.

#### Linux & macOS

Add to `~/.bashrc`, `~/.zshrc`, or your shell's profile:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then reload your shell:

```bash
source ~/.bashrc  # or ~/.zshrc
```

#### Windows

The installation script provides three options:
1. **PowerShell**: `$env:PATH = 'C:\...\TunGo\bin;' + $env:PATH`
2. **Command Prompt**: `setx PATH "C:\...\TunGo\bin;%PATH%"`
3. **GUI**: Windows Settings > Environment Variables

### Verify Installation

```bash
tungo --help
```

## Environment Variables

### Linux & macOS

- `TUNGO_INSTALL_DIR` - Custom installation directory (default: `~/.local/bin`)

### Windows

- `-InstallDir` parameter - Custom installation directory (default: `%APPDATA%\TunGo\bin`)

## Supported Platforms

### Linux
- ✅ x86_64 (amd64)
- ✅ arm64 (aarch64)

### macOS
- ✅ Intel (amd64)
- ✅ Apple Silicon (arm64)

### Windows
- ✅ x86_64 (amd64)
- ✅ ARM64

## Updating

To update to the latest version, simply run the installation script again:

```bash
./scripts/install.sh
```

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

## Troubleshooting

### "Command not found" or "not recognized"

The installation directory is not in your PATH. Follow the "Add to PATH" section above.

### Permission denied on Linux/macOS

Make the script executable:

```bash
chmod +x scripts/install.sh
```

### PowerShell execution policy error on Windows

Run with `-ExecutionPolicy Bypass`:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

### Download fails

- Check your internet connection
- Verify GitHub is accessible
- Try downloading manually from [releases page](https://github.com/sombochea/tungo/releases)

## Support

For issues, visit: https://github.com/sombochea/tungo/issues
