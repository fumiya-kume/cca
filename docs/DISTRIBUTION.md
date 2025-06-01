# ccAgents Distribution Guide

This document provides comprehensive information about ccAgents distribution, packaging, and release management.

## Distribution Overview

ccAgents is distributed through multiple channels to ensure easy installation and updates across all supported platforms:

- **GitHub Releases**: Primary distribution with platform-specific binaries
- **Homebrew** (macOS): Package manager integration
- **Installation Scripts**: Automated installation for Unix-like systems
- **PowerShell Script**: Automated installation for Windows
- **Manual Installation**: Direct binary download and setup

## Supported Platforms

| Platform | Architecture | Binary Format | Installation Method |
|----------|--------------|---------------|-------------------|
| Linux | amd64 | tar.gz | Install script, Manual |
| Linux | arm64 | tar.gz | Install script, Manual |
| macOS | amd64 | tar.gz | Homebrew, Install script, Manual |
| macOS | arm64 | tar.gz | Homebrew, Install script, Manual |
| Windows | amd64 | zip | PowerShell script, Manual |
| Windows | arm64 | zip | PowerShell script, Manual |

## Installation Methods

### 1. Quick Install (Recommended)

**Unix-like systems (Linux/macOS):**
```bash
curl -sSL https://raw.githubusercontent.com/fumiya-kume/cca/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/fumiya-kume/cca/main/install.ps1 | iex
```

### 2. Package Managers

**Homebrew (macOS):**
```bash
brew install fumiya-kume/tap/ccagents
```

**Scoop (Windows):**
```powershell
# Coming soon
scoop bucket add fumiya-kume https://github.com/fumiya-kume/scoop-bucket
scoop install ccagents
```

### 3. Manual Installation

1. Download the appropriate binary from [GitHub Releases](https://github.com/fumiya-kume/cca/releases)
2. Extract the archive
3. Move the binary to a directory in your PATH
4. Make it executable (Unix-like systems): `chmod +x ccagents`

## Build System

### Cross-Platform Builds

The build system supports multiple platforms and architectures:

```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux    # Linux (amd64 + arm64)
make build-darwin   # macOS (amd64 + arm64)
make build-windows  # Windows (amd64 + arm64)

# Create release archives
make archives

# Generate checksums
make checksums

# Complete release preparation
make release-prep
```

### Build Configuration

Build variables are automatically set:
- `version`: Git tag or commit hash
- `commit`: Git commit SHA
- `buildDate`: UTC build timestamp

These are embedded in the binary for version reporting.

## Release Process

### Automated Releases

Releases are automated through GitHub Actions:

1. **Tag Creation**: Push a version tag (e.g., `v1.0.0`)
2. **Build**: Cross-platform binaries are built automatically
3. **Testing**: Comprehensive testing on all platforms
4. **Packaging**: Archives and checksums are generated
5. **Release**: GitHub release is created with assets
6. **Homebrew**: Formula is automatically updated

### Manual Release Process

For manual releases or testing:

```bash
# 1. Prepare release
make release-prep

# 2. Verify artifacts
./scripts/verify-release.sh

# 3. Test installation
./install.sh --check

# 4. Generate changelog
./scripts/generate-changelog.sh --version v1.0.0

# 5. Create GitHub release manually
gh release create v1.0.0 ./dist/* --title "ccAgents v1.0.0" --notes-file CHANGELOG.md
```

## Security and Verification

### Checksums

All release artifacts include SHA256 checksums:
- Generated automatically during build
- Verified by installation scripts
- Available in `checksums.txt` with each release

### Signature Verification

Release artifacts are signed for security:
```bash
# Verify checksums
cd dist/
sha256sum -c checksums.txt

# Manual checksum verification
sha256sum ccagents-linux-amd64.tar.gz
```

### Installation Script Security

Installation scripts include security measures:
- HTTPS-only downloads
- Checksum verification
- Secure temporary directories
- Permission validation

## Update Mechanism

ccAgents includes a built-in update mechanism:

```bash
# Check for updates
ccagents update --check

# Update to latest version
ccagents update

# Update to specific version
ccagents update --version v1.2.0

# Force update
ccagents update --force
```

### Update Process

1. **Version Check**: Compare current vs. latest release
2. **Download**: Fetch appropriate binary for platform
3. **Verify**: Validate checksums and signatures
4. **Install**: Replace current binary atomically
5. **Verify**: Confirm successful update

## Distribution Testing

### Verification Script

Use the verification script to test releases:

```bash
# Verify latest release
./scripts/verify-release.sh

# Verify specific version
./scripts/verify-release.sh --version v1.0.0
```

### Test Matrix

The verification process includes:
- ✅ Binary availability for all platforms
- ✅ Archive integrity and extraction
- ✅ Checksum verification
- ✅ Basic functionality testing
- ✅ Installation script validation
- ✅ Update mechanism testing

## Package Management Integration

### Homebrew Formula

The Homebrew formula is automatically maintained:
- Located at `Formula/ccagents.rb`
- Updated by GitHub Actions on each release
- Includes shell completion and dependencies

### Future Package Managers

Planned integrations:
- **Scoop** (Windows): Package manager support
- **Chocolatey** (Windows): Package repository
- **Snap** (Linux): Universal package format
- **Flatpak** (Linux): Application distribution
- **Docker**: Containerized distribution

## Distribution Analytics

### Download Tracking

GitHub Releases provides automatic download analytics:
- Download counts per platform
- Release adoption rates
- Geographic distribution

### Telemetry (Optional)

ccAgents includes optional, privacy-respecting telemetry:
- Usage patterns (if enabled)
- Error reporting (anonymized)
- Performance metrics
- Feature usage statistics

*Note: All telemetry is opt-in and can be disabled.*

## Release Versioning

### Semantic Versioning

ccAgents follows [Semantic Versioning](https://semver.org/):
- **MAJOR** (X.0.0): Breaking changes
- **MINOR** (0.X.0): New features, backward compatible
- **PATCH** (0.0.X): Bug fixes, backward compatible

### Pre-release Versions

Development versions include:
- **Alpha** (v1.0.0-alpha.1): Early development
- **Beta** (v1.0.0-beta.1): Feature complete, testing
- **RC** (v1.0.0-rc.1): Release candidate

## Troubleshooting Distribution

### Common Issues

**Download failures:**
- Check internet connectivity
- Verify GitHub access
- Try alternative download methods

**Installation permission errors:**
- Use `sudo` for system-wide installation
- Install to user directory
- Check file permissions

**Binary not found after installation:**
- Verify PATH configuration
- Restart terminal session
- Check installation directory

**Update failures:**
- Check write permissions to binary location
- Ensure ccAgents is not running
- Try manual installation

### Support Channels

- [GitHub Issues](https://github.com/fumiya-kume/cca/issues): Bug reports
- [GitHub Discussions](https://github.com/fumiya-kume/cca/discussions): Questions
- [Documentation](https://github.com/fumiya-kume/cca/docs): Comprehensive guides

## Contributing to Distribution

### Adding New Platforms

To add support for new platforms:

1. Update `Makefile` with new platform targets
2. Modify GitHub Actions workflow
3. Update installation scripts
4. Add platform-specific testing
5. Update documentation

### Improving Package Managers

Help us integrate with more package managers:
- Create package definitions
- Test installation processes
- Maintain update automation
- Documentation updates

## Distribution Roadmap

### Current Status
- ✅ Cross-platform binaries
- ✅ GitHub Releases automation
- ✅ Installation scripts
- ✅ Homebrew integration
- ✅ Update mechanism
- ✅ Security verification

### Planned Improvements
- 🔄 Scoop package (Windows)
- 🔄 Snap package (Linux)
- 🔄 Docker containers
- 🔄 Package signing with GPG
- 🔄 Mirror repositories
- 🔄 CDN distribution

---

For the latest distribution information, visit the [ccAgents GitHub repository](https://github.com/fumiya-kume/cca).