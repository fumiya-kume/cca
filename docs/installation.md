# Installation Guide

This guide covers all installation methods for ccAgents on different platforms.

## System Requirements

### Minimum Requirements
- **Operating System**: macOS 10.15+, Linux (Ubuntu 18.04+, CentOS 7+), Windows 10+
- **Memory**: 512 MB RAM
- **Storage**: 100 MB free space
- **Go**: 1.21 or later (for source installation)

### Recommended Requirements
- **Memory**: 2 GB RAM
- **Storage**: 500 MB free space
- **CPU**: Multi-core processor for better performance

## Prerequisites

Before installing ccAgents, ensure you have the following tools installed and configured:

### 1. Git
ccAgents requires Git for repository operations.

**macOS:**
```bash
# Using Homebrew
brew install git

# Using Xcode Command Line Tools
xcode-select --install
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt update
sudo apt install git
```

**Linux (CentOS/RHEL):**
```bash
sudo yum install git
```

**Windows:**
- Download from [git-scm.com](https://git-scm.com/download/win)
- Or use [Chocolatey](https://chocolatey.org/): `choco install git`

### 2. GitHub CLI (gh)
ccAgents uses GitHub CLI for GitHub operations.

**macOS:**
```bash
brew install gh
```

**Linux:**
```bash
# Ubuntu/Debian
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update
sudo apt install gh

# CentOS/RHEL
sudo dnf install 'dnf-command(config-manager)'
sudo dnf config-manager --add-repo https://cli.github.com/packages/rpm/gh-cli.repo
sudo dnf install gh
```

**Windows:**
```bash
# Using Chocolatey
choco install gh

# Using Scoop
scoop bucket add github-gh https://github.com/cli/scoop-gh.git
scoop install gh
```

### 3. Claude Code CLI
ccAgents requires Claude Code for AI operations.

```bash
# Install Claude Code CLI
curl -sSL https://claude.ai/cli/install.sh | bash

# Or follow instructions at https://docs.anthropic.com/claude/docs/cli
```

## Installation Methods

### Method 1: Binary Releases (Recommended)

This is the easiest method for most users.

**macOS and Linux:**
```bash
# Download and install the latest release
curl -sSL https://github.com/fumiya-kume/cca/releases/latest/download/install.sh | bash

# Or manually download
curl -L https://github.com/fumiya-kume/cca/releases/latest/download/ccagents-$(uname -s)-$(uname -m).tar.gz | tar xz
sudo mv ccagents /usr/local/bin/
```

**Windows:**
```powershell
# Download the Windows binary
Invoke-WebRequest -Uri "https://github.com/fumiya-kume/cca/releases/latest/download/ccagents-windows-amd64.zip" -OutFile "ccagents.zip"
Expand-Archive -Path "ccagents.zip" -DestinationPath "C:\\Program Files\\ccagents"
# Add C:\Program Files\ccagents to your PATH
```

### Method 2: Package Managers

**macOS (Homebrew):**
```bash
brew tap fumiya-kume/ccagents
brew install ccagents
```

**Linux (Snap):**
```bash
sudo snap install ccagents
```

**Arch Linux (AUR):**
```bash
yay -S ccagents-bin
# or
paru -S ccagents-bin
```

### Method 3: Go Install

If you have Go installed:

```bash
go install github.com/fumiya-kume/cca/cmd/ccagents@latest
```

### Method 4: Build from Source

For developers or users who want the latest features:

```bash
# Clone the repository
git clone https://github.com/fumiya-kume/cca.git
cd cca

# Build and install
make build
sudo make install

# Or manually
go build -o ccagents cmd/ccagents/main.go
sudo mv ccagents /usr/local/bin/
```

## Initial Configuration

After installation, you need to configure ccAgents:

### 1. Verify Installation

```bash
ccagents version
```

### 2. Initialize Configuration

```bash
ccagents init
```

This creates a default configuration file at `~/.ccagents.yaml`.

### 3. Configure GitHub Authentication

```bash
# Authenticate with GitHub
gh auth login

# Verify authentication
gh auth status
```

### 4. Configure Claude Code

```bash
# Authenticate with Claude
claude auth

# Verify authentication
claude config list
```

### 5. Test Installation

```bash
# Run a quick test
ccagents validate

# Check system status
ccagents status
```

## Configuration File

The default configuration file is created at `~/.ccagents.yaml`:

```yaml
version: "1.0"

claude:
  model: "claude-3-sonnet-20240229"
  max_tokens: 4096
  temperature: 0.1

github:
  api_version: "2022-11-28"
  timeout: 30s
  max_retries: 3

workflow:
  enable_auto_merge: false
  require_review: true
  run_tests: true
  security_scan: true

ui:
  theme: "auto"
  show_progress: true
  enable_notifications: true

logging:
  level: "info"
  format: "pretty"
  output: "stdout"
```

## Environment Variables

ccAgents can be configured using environment variables:

```bash
export CCAGENTS_CONFIG_PATH="~/.ccagents.yaml"
export CCAGENTS_LOG_LEVEL="debug"
export CCAGENTS_GITHUB_TOKEN="your_github_token"
export CCAGENTS_CLAUDE_API_KEY="your_claude_api_key"
```

## Troubleshooting Installation

### Common Issues

**Issue**: `ccagents: command not found`
```bash
# Solution: Check if the binary is in your PATH
echo $PATH
which ccagents

# Add to PATH if needed (add to ~/.bashrc or ~/.zshrc)
export PATH="/usr/local/bin:$PATH"
```

**Issue**: Permission denied
```bash
# Solution: Make the binary executable
chmod +x /usr/local/bin/ccagents

# Or install without sudo
mkdir -p ~/bin
mv ccagents ~/bin/
export PATH="$HOME/bin:$PATH"
```

**Issue**: GitHub authentication fails
```bash
# Solution: Reconfigure GitHub CLI
gh auth logout
gh auth login --web
```

**Issue**: Claude Code authentication fails
```bash
# Solution: Check Claude configuration
claude auth status
claude auth refresh
```

**Issue**: Configuration validation errors
```bash
# Solution: Reset configuration
ccagents init --force

# Or validate current configuration
ccagents validate --config ~/.ccagents.yaml
```

### Getting Help

If you encounter issues:

1. **Check the logs**: `ccagents logs --level debug`
2. **Validate configuration**: `ccagents validate`
3. **Check system status**: `ccagents status`
4. **View documentation**: [docs.ccagents.dev](https://docs.ccagents.dev)
5. **Create an issue**: [GitHub Issues](https://github.com/fumiya-kume/cca/issues)

## Uninstallation

To remove ccAgents:

**Binary installation:**
```bash
sudo rm /usr/local/bin/ccagents
rm -rf ~/.ccagents.yaml
rm -rf ~/.ccagents/
```

**Homebrew:**
```bash
brew uninstall ccagents
brew untap fumiya-kume/ccagents
```

**Go install:**
```bash
go clean -i github.com/fumiya-kume/cca/cmd/ccagents
```

## Next Steps

After successful installation:

1. Read the [Getting Started Guide](getting-started.md)
2. Explore [Configuration Examples](examples/)
3. Set up your first automation workflow
4. Join the [community discussions](https://github.com/fumiya-kume/cca/discussions)

## Platform-Specific Notes

### macOS
- ccAgents works on both Intel and Apple Silicon Macs
- Gatekeeper may require approval for unsigned binaries
- Use `sudo spctl --master-disable` temporarily if needed

### Linux
- Works on most distributions with systemd
- Requires GLIBC 2.17+ (most modern distributions)
- WSL is supported on Windows

### Windows
- Requires Windows 10 version 1903 or later
- PowerShell 5.1+ recommended
- WSL2 provides the best experience

## Security Considerations

- ccAgents stores credentials securely using the system keychain
- API keys are encrypted at rest
- Network traffic is encrypted with TLS
- Audit logs are available for compliance

For more security information, see the [Security Guide](security.md).