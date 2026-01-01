# TunGo Client Installation Script for Windows
# Run in PowerShell as regular user (no admin required)
# Usage: powershell -ExecutionPolicy Bypass -File scripts/install.ps1

param(
    [string]$InstallDir = "$env:APPDATA\TunGo\bin"
)

# Color output functions
function Write-Error-Custom {
    param([string]$Message)
    Write-Host "✗ Error: $Message" -ForegroundColor Red
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Cyan
}

function Write-Warning-Custom {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor Yellow
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-Error-Custom "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Get latest release version
function Get-LatestVersion {
    Write-Info "Fetching latest release information..."
    
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/sombochea/tungo/releases/latest" -ErrorAction Stop
        $version = $response.tag_name
        
        if ([string]::IsNullOrEmpty($version)) {
            Write-Error-Custom "Could not fetch latest version from GitHub"
            exit 1
        }
        
        return $version
    }
    catch {
        Write-Error-Custom "Failed to fetch release information: $_"
        exit 1
    }
}

# Download binary
function Download-Binary {
    param(
        [string]$Version,
        [string]$Architecture
    )
    
    $downloadUrl = "https://github.com/sombochea/tungo/releases/download/$Version/tungo-windows-$Architecture.exe"
    $tempFile = Join-Path $env:TEMP "tungo-download.exe"
    
    Write-Info "Downloading tungo $Version for Windows/$Architecture..."
    
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -ErrorAction Stop
        Write-Success "Download completed"
        return $tempFile
    }
    catch {
        Write-Error-Custom "Failed to download binary: $_"
        exit 1
    }
}

# Install binary
function Install-Binary {
    param(
        [string]$SourcePath,
        [string]$DestinationDir
    )
    
    # Create installation directory if it doesn't exist
    if (-not (Test-Path $DestinationDir)) {
        Write-Info "Creating installation directory: $DestinationDir"
        $null = New-Item -ItemType Directory -Path $DestinationDir -Force
    }
    
    $targetPath = Join-Path $DestinationDir "tungo.exe"
    
    Write-Info "Installing binary to $targetPath..."
    
    try {
        Copy-Item -Path $SourcePath -Destination $targetPath -Force
        Remove-Item -Path $SourcePath -Force
        Write-Success "Binary installed successfully"
        return $targetPath
    }
    catch {
        Write-Error-Custom "Failed to install binary: $_"
        exit 1
    }
}

# Check if directory is in PATH
function Test-PathVariable {
    param([string]$Directory)
    
    $paths = $env:PATH -split ';'
    return $paths -contains $Directory
}

# Suggest PATH update
function Show-PathUpdateGuide {
    param([string]$Directory)
    
    if (-not (Test-PathVariable $Directory)) {
        Write-Warning-Custom "Installation directory is not in your PATH"
        Write-Host ""
        Write-Host "To use 'tungo' command from anywhere, add the directory to PATH:"
        Write-Host ""
        Write-Host "Option 1: Using PowerShell (Recommended)"
        Write-Host "  `$env:PATH = '$Directory;' + `$env:PATH"
        Write-Host ""
        Write-Host "Option 2: Using Command Prompt"
        Write-Host "  setx PATH ""$Directory;%PATH%"""
        Write-Host ""
        Write-Host "Option 3: Using Windows Settings"
        Write-Host "  1. Press Win+X, search 'Environment Variables'"
        Write-Host "  2. Click 'Edit the system environment variables'"
        Write-Host "  3. Click 'Environment Variables' button"
        Write-Host "  4. Under 'User variables', click 'New'"
        Write-Host "  5. Variable name: PATH, Value: $Directory"
        Write-Host ""
        Write-Host "After adding to PATH, restart PowerShell for changes to take effect."
        Write-Host ""
    }
}

# Main installation
function Main {
    Write-Host ""
    Write-Host "╔════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║   TunGo Client Installation Script     ║" -ForegroundColor Cyan
    Write-Host "╚════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    
    # Detect architecture
    $architecture = Get-Architecture
    Write-Info "Detected architecture: Windows/$architecture"
    
    # Get latest version
    $version = Get-LatestVersion
    Write-Info "Latest version: $version"
    
    # Download binary
    $tempFile = Download-Binary -Version $version -Architecture $architecture
    
    # Install binary
    $installPath = Install-Binary -SourcePath $tempFile -DestinationDir $InstallDir
    
    # Verify installation
    Write-Info "Verifying installation..."
    try {
        $output = & $installPath --version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Installation verified"
        } else {
            Write-Warning-Custom "Binary verification returned non-zero exit code, but installation completed"
        }
    }
    catch {
        Write-Warning-Custom "Could not verify binary, but installation completed"
    }
    
    # Print summary
    Write-Host ""
    Write-Host "╔════════════════════════════════════════╗" -ForegroundColor Green
    Write-Host "║   Installation Complete!              ║" -ForegroundColor Green
    Write-Host "╚════════════════════════════════════════╝" -ForegroundColor Green
    Write-Host ""
    
    Write-Host "Installation Details:"
    Write-Host "  Version: $version"
    Write-Host "  Platform: Windows/$architecture"
    Write-Host "  Location: $installPath"
    Write-Host ""
    
    # Check PATH
    Show-PathUpdateGuide $InstallDir
    
    Write-Host "Next steps:"
    Write-Host "  1. Add installation directory to PATH (if needed)"
    Write-Host "  2. Restart PowerShell"
    Write-Host "  3. Run: tungo --help"
    Write-Host ""
}

# Run main
Main
