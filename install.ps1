# ccAgents Windows Installation Script
# This script downloads and installs the latest version of ccAgents on Windows

param(
    [string]$Version = "",
    [string]$InstallDir = "",
    [switch]$Help
)

# Constants
$REPO = "fumiya-kume/cca"
$BINARY_NAME = "ccagents.exe"
$DEFAULT_INSTALL_DIR = "$env:LOCALAPPDATA\ccAgents\bin"

# Color functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
    exit 1
}

# Show help
if ($Help) {
    Write-Host "ccAgents Windows Installation Script"
    Write-Host ""
    Write-Host "Usage: .\install.ps1 [options]"
    Write-Host ""
    Write-Host "Options:"
    Write-Host "  -Help           Show this help message"
    Write-Host "  -Version        Install specific version (e.g., v1.0.0)"
    Write-Host "  -InstallDir     Install to specific directory"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\install.ps1                                    # Install latest version"
    Write-Host "  .\install.ps1 -Version v1.0.0                   # Install specific version"
    Write-Host "  .\install.ps1 -InstallDir C:\Tools\ccAgents     # Install to specific directory"
    exit 0
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { Write-Error "Unsupported architecture: $arch" }
    }
}

# Get latest release version
function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"
        return $response.tag_name
    }
    catch {
        Write-Error "Failed to get latest version information: $_"
    }
}

# Download file
function Download-File {
    param(
        [string]$Url,
        [string]$Output
    )
    
    Write-Info "Downloading from: $Url"
    
    try {
        Invoke-WebRequest -Uri $Url -OutFile $Output -UseBasicParsing
    }
    catch {
        Write-Error "Failed to download file: $_"
    }
}

# Setup installation directory
function Setup-InstallDir {
    param([string]$Dir)
    
    if ($Dir -eq "") {
        $Dir = $DEFAULT_INSTALL_DIR
    }
    
    if (!(Test-Path $Dir)) {
        Write-Info "Creating installation directory: $Dir"
        New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    }
    
    Write-Info "Installation directory: $Dir"
    return $Dir
}

# Check if directory is in PATH
function Test-PathContains {
    param([string]$Dir)
    
    $pathDirs = $env:PATH -split ';'
    return $pathDirs -contains $Dir
}

# Add directory to PATH
function Add-ToPath {
    param([string]$Dir)
    
    if (!(Test-PathContains $Dir)) {
        Write-Warning "$Dir is not in your PATH"
        Write-Warning "Adding $Dir to your PATH..."
        
        # Get current user PATH
        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($userPath -eq $null) {
            $userPath = ""
        }
        
        # Add new directory
        $newPath = if ($userPath -eq "") { $Dir } else { "$userPath;$Dir" }
        
        try {
            [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
            Write-Success "Added $Dir to your PATH"
            Write-Warning "Please restart your terminal or run: `$env:PATH = `"$Dir;`$env:PATH`""
        }
        catch {
            Write-Warning "Failed to add $Dir to PATH automatically"
            Write-Warning "Please add it manually through System Properties > Environment Variables"
        }
    }
}

# Extract zip file
function Extract-Archive {
    param(
        [string]$ArchivePath,
        [string]$DestinationPath
    )
    
    Write-Info "Extracting archive..."
    
    try {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($ArchivePath, $DestinationPath)
    }
    catch {
        Write-Error "Failed to extract archive: $_"
    }
}

# Main installation function
function Install-ccAgents {
    Write-Info "Starting ccAgents installation..."
    
    # Detect architecture
    $arch = Get-Architecture
    Write-Info "Detected architecture: $arch"
    
    # Get version
    $installVersion = if ($Version -eq "") { Get-LatestVersion } else { $Version }
    if ($installVersion -eq "") {
        Write-Error "Failed to determine version to install"
    }
    Write-Info "Installing version: $installVersion"
    
    # Setup installation directory
    $finalInstallDir = Setup-InstallDir $InstallDir
    
    # Create temporary directory
    $tmpDir = Join-Path $env:TEMP "ccagents-install"
    if (Test-Path $tmpDir) {
        Remove-Item $tmpDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
    
    # Construct download URL
    $archiveName = "ccagents-windows-$arch.zip"
    $downloadUrl = "https://github.com/$REPO/releases/download/$installVersion/$archiveName"
    $archivePath = Join-Path $tmpDir $archiveName
    
    # Download the archive
    Download-File $downloadUrl $archivePath
    
    # Extract the archive
    $extractPath = Join-Path $tmpDir "extracted"
    Extract-Archive $archivePath $extractPath
    
    # Find the binary
    $binaryPath = Join-Path $extractPath "ccagents-windows-$arch.exe"
    if (!(Test-Path $binaryPath)) {
        Write-Error "Binary not found in archive: $binaryPath"
    }
    
    # Install the binary
    Write-Info "Installing to $finalInstallDir..."
    $finalBinaryPath = Join-Path $finalInstallDir $BINARY_NAME
    Copy-Item $binaryPath $finalBinaryPath -Force
    
    # Cleanup
    Remove-Item $tmpDir -Recurse -Force
    
    # Add to PATH if needed
    Add-ToPath $finalInstallDir
    
    # Verify installation
    try {
        $installedVersion = & $finalBinaryPath version --short 2>$null
        Write-Success "ccAgents installed successfully!"
        Write-Success "Version: $installedVersion"
        Write-Success "Location: $finalBinaryPath"
    }
    catch {
        Write-Success "ccAgents installed to: $finalBinaryPath"
        Write-Info "You may need to restart your terminal or add the installation directory to your PATH"
    }
    
    Write-Info "Run 'ccagents help' to get started!"
}

# Check PowerShell version
if ($PSVersionTable.PSVersion.Major -lt 3) {
    Write-Error "PowerShell 3.0 or later is required"
}

# Run installation
try {
    Install-ccAgents
}
catch {
    Write-Error "Installation failed: $_"
}