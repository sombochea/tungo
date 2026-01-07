# TunGo Client Installation Script for Windows
# Run in PowerShell as regular user (no admin required)
# Usage: powershell -ExecutionPolicy Bypass -File install.ps1

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\TunGo\bin",
    [string]$Version = ""
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Color output functions
function Write-Error-Custom {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Write-Success {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function Write-Warning-Custom {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            throw "Unsupported architecture: $arch"
        }
    }
}

# Get latest release version (CLI only, excluding SDK releases)
function Get-LatestVersion {
    Write-Info "Fetching latest release information..."
    
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/sombochea/tungo/releases" -ErrorAction Stop
        # Filter out SDK releases (those starting with "sdk-")
        $version = $response | 
            Where-Object { $_.tag_name -notmatch '^sdk-' } | 
            Select-Object -First 1 -ExpandProperty tag_name
        
        if ([string]::IsNullOrEmpty($version)) {
            throw "Could not fetch latest CLI version from GitHub"
        }
        
        return $version
    }
    catch {
        throw "Failed to fetch release information: $_"
    }
}

# Download binary
function Download-Binary {
    param(
        [string]$Version,
        [string]$Architecture
    )
    
    $binaryName = "tungo-windows-$Architecture.exe"
    $downloadUrl = "https://github.com/sombochea/tungo/releases/download/$Version/$binaryName"
    $tempFile = Join-Path $env:TEMP "tungo-$Version-$Architecture.exe"
    
    Write-Info "Download URL: $downloadUrl"
    Write-Info "Downloading tungo $Version for Windows/$Architecture..."
    Write-Info "Temporary file: $tempFile"
    
    try {
        # Remove old temp file if exists
        if (Test-Path $tempFile) {
            Remove-Item -Path $tempFile -Force
        }
        
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -ErrorAction Stop
        
        # Verify file was downloaded
        if (-not (Test-Path $tempFile)) {
            throw "Download completed but file not found at $tempFile"
        }
        
        $fileSize = (Get-Item $tempFile).Length
        if ($fileSize -eq 0) {
            throw "Downloaded file is empty"
        }
        
        $sizeInMB = [math]::Round($fileSize/1MB, 2)
        Write-Success "Download completed - Size: $sizeInMB MB"
        return $tempFile
    }
    catch {
        # Check if release exists for better error message
        try {
            $releaseCheck = Invoke-RestMethod -Uri "https://api.github.com/repos/sombochea/tungo/releases/tags/$Version" -ErrorAction Stop
            throw "Release $Version exists, but binary $binaryName might not be uploaded yet. Error: $_"
        }
        catch {
            throw "Failed to download binary from $downloadUrl. Error: $_"
        }
    }
}

# Install binary
function Install-Binary {
    param(
        [string]$SourcePath,
        [string]$DestinationDir
    )
    
    Write-Info "Installation directory: $DestinationDir"
    
    # Create installation directory if it doesn't exist
    if (-not (Test-Path $DestinationDir)) {
        Write-Info "Creating installation directory..."
        try {
            $null = New-Item -ItemType Directory -Path $DestinationDir -Force -ErrorAction Stop
            Write-Success "Directory created: $DestinationDir"
        }
        catch {
            throw "Failed to create directory: $_"
        }
    } else {
        Write-Info "Installation directory already exists"
    }
    
    $targetPath = Join-Path $DestinationDir "tungo.exe"
    
    # Check if source file exists
    if (-not (Test-Path $SourcePath)) {
        throw "Source file not found: $SourcePath"
    }
    
    Write-Info "Installing binary..."
    Write-Info "  From: $SourcePath"
    Write-Info "  To:   $targetPath"
    
    try {
        # Remove existing binary if present
        if (Test-Path $targetPath) {
            Write-Info "Removing existing binary..."
            Remove-Item -Path $targetPath -Force -ErrorAction Stop
        }
        
        # Copy new binary
        Copy-Item -Path $SourcePath -Destination $targetPath -Force -ErrorAction Stop
        
        # Verify installation
        if (-not (Test-Path $targetPath)) {
            throw "File was not copied to destination"
        }
        
        $installedSize = (Get-Item $targetPath).Length
        Write-Success "Binary installed successfully - Size: $([math]::Round($installedSize/1MB, 2)) MB"
        
        # Clean up temp file
        if (Test-Path $SourcePath) {
            Remove-Item -Path $SourcePath -Force -ErrorAction SilentlyContinue
        }
        
        return $targetPath
    }
    catch {
        throw "Failed to install binary: $_"
    }
}

# Check if directory is in PATH
function Test-PathVariable {
    param([string]$Directory)
    
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    $machinePath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
    $currentPath = $env:PATH
    
    $allPaths = ($userPath + ";" + $machinePath + ";" + $currentPath) -split ';' | Where-Object { $_ -ne "" }
    
    foreach ($path in $allPaths) {
        if ($path.TrimEnd('\') -eq $Directory.TrimEnd('\')) {
            return $true
        }
    }
    
    return $false
}

# Add directory to PATH
function Add-ToPath {
    param([string]$Directory)
    
    try {
        $currentUserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        
        if ([string]::IsNullOrEmpty($currentUserPath)) {
            $newPath = $Directory
        } else {
            $newPath = "$currentUserPath;$Directory"
        }
        
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        $env:PATH = "$Directory;$env:PATH"
        
        Write-Host "[OK] Added to PATH successfully" -ForegroundColor Green
        Write-Host "[INFO] You may need to restart your terminal for changes to take full effect" -ForegroundColor Cyan
        return $true
    }
    catch {
        Write-Host "[WARN] Failed to add to PATH automatically: $_" -ForegroundColor Yellow
        return $false
    }
}

# Suggest PATH update
function Show-PathUpdateGuide {
    param(
        [string]$Directory,
        [string]$BinaryPath
    )
    
    # Verify binary exists before proceeding
    if (-not (Test-Path $BinaryPath)) {
        Write-Host ""
        Write-Host "[ERROR] Binary not found at: $BinaryPath" -ForegroundColor Red
        Write-Host "Installation may have failed. Please try again." -ForegroundColor Yellow
        return
    }
    
    Write-Host ""
    $isInPath = Test-PathVariable $Directory
    
    if (-not $isInPath) {
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host "  PATH Setup Required" -ForegroundColor Yellow
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Would you like to add TunGo to your PATH automatically? (Y/n): " -NoNewline -ForegroundColor Cyan
        $response = Read-Host
        Write-Host ""
        
        if ($response -eq "" -or $response -eq "Y" -or $response -eq "y") {
            Write-Host "[INFO] Adding to PATH..." -ForegroundColor Cyan
            $added = Add-ToPath $Directory
            
            if ($added) {
                Write-Host ""
                Write-Host "[OK] PATH updated! You can now use 'tungo' command." -ForegroundColor Green
                Write-Host ""
                Write-Host "Test it by running:" -ForegroundColor Cyan
                Write-Host "  tungo --version" -ForegroundColor White
            } else {
                Write-Host ""
                Write-Host "[WARN] Automatic PATH update failed. Please add manually." -ForegroundColor Yellow
                Show-ManualPathInstructions $Directory
            }
        } else {
            Write-Host "[INFO] Skipped automatic PATH setup." -ForegroundColor Cyan
            Show-ManualPathInstructions $Directory
        }
    } else {
        Write-Host "[OK] Installation directory is already in PATH" -ForegroundColor Green
        Write-Host ""
        Write-Host "You can use 'tungo' command immediately:" -ForegroundColor Cyan
        Write-Host "  tungo --version" -ForegroundColor White
    }
    
    Write-Host ""
    Write-Host "Binary location:" -ForegroundColor Cyan
    Write-Host "  $BinaryPath" -ForegroundColor White
    Write-Host ""
}

function Show-ManualPathInstructions {
    param([string]$Directory)
    
    Write-Host ""
    Write-Host "Manual PATH Setup Options:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Option 1: Quick (Current Session Only)" -ForegroundColor Cyan
    Write-Host "  `$env:PATH = '$Directory;' + `$env:PATH" -ForegroundColor White
    Write-Host ""
    Write-Host "Option 2: Permanent (Command Prompt)" -ForegroundColor Cyan
    Write-Host "  setx PATH ""%PATH%;$Directory""" -ForegroundColor White
    Write-Host ""
    Write-Host "Option 3: Via Windows Settings" -ForegroundColor Cyan
    Write-Host "  1. Press Win+X → System → Advanced system settings"
    Write-Host "  2. Click 'Environment Variables'"
    Write-Host "  3. Under 'User variables', edit PATH"
    Write-Host "  4. Add: $Directory"
    Write-Host ""
}

# Main installation
function Main {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "   TunGo Client Installation Script    " -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""
    
    # Detect architecture
    try {
        $architecture = Get-Architecture
        Write-Info "Detected architecture: Windows/$architecture"
    }
    catch {
        throw "Failed to detect architecture: $_"
    }
    
    # Get version
    try {
        if ([string]::IsNullOrEmpty($Version)) {
            $version = Get-LatestVersion
            Write-Info "Latest version: $version"
        } else {
            $version = $Version
            Write-Info "Installing version: $version"
        }
    }
    catch {
        throw "Failed to get version: $_"
    }
    
    # Download binary
    try {
        $tempFile = Download-Binary -Version $version -Architecture $architecture
        
        # Validate tempFile was returned
        if ([string]::IsNullOrEmpty($tempFile)) {
            throw "Download-Binary did not return a valid path"
        }
        
        # Verify downloaded file
        if (-not (Test-Path $tempFile)) {
            throw "Downloaded file not found: $tempFile"
        }
    }
    catch {
        throw "Failed to download: $_"
    }
    
    # Install binary
    try {
        $installPath = Install-Binary -SourcePath $tempFile -DestinationDir $InstallDir
        
        # Validate installPath was returned
        if ([string]::IsNullOrEmpty($installPath)) {
            throw "Install-Binary did not return a valid path"
        }
        
        if (-not (Test-Path $installPath)) {
            throw "Install-Binary returned path that doesn't exist: $installPath"
        }
    }
    catch {
        throw "Failed to install: $_"
    }
    
    # Verify installation
    Write-Info "Verifying installation..."
    if (-not (Test-Path $installPath)) {
        throw "Binary not found after installation: $installPath"
    }
    
    # Verify file size
    $installedSize = (Get-Item $installPath).Length
    if ($installedSize -eq 0) {
        Remove-Item -Path $installPath -Force -ErrorAction SilentlyContinue
        throw "Installed binary is empty - 0 bytes"
    }
    
    $sizeInMB = [math]::Round($installedSize/1MB, 2)
    Write-Success "Binary file verified - Size: $sizeInMB MB"
    
    try {
        $versionOutput = & $installPath --version 2>&1 | Out-String
        if ($null -ne $versionOutput -and $versionOutput.Length -gt 0) {
            $trimmedOutput = $versionOutput.Trim()
            if ($trimmedOutput.Length -gt 0) {
                Write-Success "Installation verified: $trimmedOutput"
            } else {
                Write-Success "Installation verified - Binary is executable"
            }
        } else {
            Write-Success "Installation verified - Binary is executable"
        }
    }
    catch {
        Write-Warning-Custom "Could not verify binary version, but file exists at: $installPath"
        Write-Host "You may need to run: tungo --version" -ForegroundColor Cyan
    }
    
    # Print summary
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "   Installation Complete!              " -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    
    Write-Host "Installation Details:" -ForegroundColor Cyan
    Write-Host "  Version:  $version"
    Write-Host "  Platform: Windows/$architecture"
    Write-Host "  Location: $installPath"
    Write-Host ""
    
    # Handle PATH
    Show-PathUpdateGuide -Directory $InstallDir -BinaryPath $installPath
    
    Write-Host ""
    Write-Host "Next Steps:" -ForegroundColor Cyan
    Write-Host "  1. Open a new PowerShell window (if PATH was just added)"
    Write-Host "  2. Run: tungo --help"
    Write-Host "  3. Start tunneling: tungo --server-url wss://your-server.com"
    Write-Host ""
    Write-Host "Documentation: https://github.com/sombochea/tungo" -ForegroundColor Cyan
    Write-Host ""
}

# Run main with error handling
try {
    Main
}
catch {
    Write-Host ""
    Write-Error-Custom "Installation failed: $_"
    Write-Host ""
    Write-Host "Stack Trace:" -ForegroundColor Red
    Write-Host $_.ScriptStackTrace
    Write-Host ""
    Write-Host "Please report this issue at: https://github.com/sombochea/tungo/issues" -ForegroundColor Yellow
    exit 1
}
