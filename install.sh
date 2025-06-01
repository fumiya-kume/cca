#!/bin/bash
# ccAgents Installation Script
# This script downloads and installs the latest version of ccAgents

set -e

# Constants
REPO="fumiya-kume/cca"
BINARY_NAME="ccagents"
INSTALL_DIR="/usr/local/bin"
TMP_DIR="/tmp/ccagents-install"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux";;
        Darwin*)    os="darwin";;
        CYGWIN*|MINGW*|MSYS*) os="windows";;
        *)          error "Unsupported operating system: $(uname -s)";;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64";;
        arm64|aarch64)  arch="arm64";;
        *)              error "Unsupported architecture: $(uname -m)";;
    esac
    
    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    if command_exists curl; then
        curl -s "https://api.github.com/repos/${REPO}/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"([^"]+)".*/\1/'
    elif command_exists wget; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | \
            grep '"tag_name":' | \
            sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "curl or wget is required to download ccAgents"
    fi
}

# Download file
download_file() {
    local url="$1"
    local output="$2"
    
    info "Downloading from: $url"
    
    if command_exists curl; then
        curl -L -o "$output" "$url"
    elif command_exists wget; then
        wget -O "$output" "$url"
    else
        error "curl or wget is required to download ccAgents"
    fi
}

# Verify installation directory
setup_install_dir() {
    # Try to use /usr/local/bin first
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ -w "$HOME/.local/bin" ]; then
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
    elif [ -w "$HOME/bin" ]; then
        INSTALL_DIR="$HOME/bin"
        mkdir -p "$INSTALL_DIR"
    else
        # Create local bin directory
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        warning "Created $INSTALL_DIR for installation"
        warning "Make sure to add $INSTALL_DIR to your PATH"
    fi
    
    info "Installation directory: $INSTALL_DIR"
}

# Check PATH
check_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warning "$INSTALL_DIR is not in your PATH"
        warning "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        warning "export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
}

# Main installation function
install_ccagents() {
    info "Starting ccAgents installation..."
    
    # Check dependencies
    if ! command_exists curl && ! command_exists wget; then
        error "Either curl or wget is required for installation"
    fi
    
    if ! command_exists tar; then
        error "tar is required for installation"
    fi
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    info "Detected platform: $platform"
    
    # Get latest version
    local version
    version=$(get_latest_version)
    if [ -z "$version" ]; then
        error "Failed to get latest version information"
    fi
    info "Latest version: $version"
    
    # Setup installation directory
    setup_install_dir
    
    # Create temporary directory
    rm -rf "$TMP_DIR"
    mkdir -p "$TMP_DIR"
    
    # Construct download URL
    local archive_name="${BINARY_NAME}-${platform}.tar.gz"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"
    local archive_path="${TMP_DIR}/${archive_name}"
    
    # Download the archive
    download_file "$download_url" "$archive_path"
    
    # Extract the archive
    info "Extracting archive..."
    cd "$TMP_DIR"
    tar -xzf "$archive_name"
    
    # Find the binary
    local binary_path="${TMP_DIR}/${BINARY_NAME}-${platform}"
    if [ ! -f "$binary_path" ]; then
        error "Binary not found in archive: $binary_path"
    fi
    
    # Install the binary
    info "Installing to $INSTALL_DIR..."
    if [ -w "$INSTALL_DIR" ]; then
        cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    else
        info "Installing requires sudo privileges..."
        sudo cp "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    fi
    
    # Cleanup
    rm -rf "$TMP_DIR"
    
    # Verify installation
    if command_exists "$BINARY_NAME"; then
        local installed_version
        installed_version=$("$BINARY_NAME" version --short 2>/dev/null || echo "unknown")
        success "ccAgents installed successfully!"
        success "Version: $installed_version"
        success "Location: $(which $BINARY_NAME)"
    else
        success "ccAgents installed to: ${INSTALL_DIR}/${BINARY_NAME}"
        check_path
    fi
    
    info "Run 'ccagents help' to get started!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "ccAgents Installation Script"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo "  --version     Install specific version (e.g., v1.0.0)"
        echo "  --dir DIR     Install to specific directory"
        echo ""
        echo "Environment Variables:"
        echo "  INSTALL_DIR   Custom installation directory"
        echo ""
        echo "Examples:"
        echo "  $0                           # Install latest version"
        echo "  $0 --version v1.0.0          # Install specific version"
        echo "  $0 --dir /usr/local/bin      # Install to specific directory"
        echo "  INSTALL_DIR=/opt/bin $0      # Install using environment variable"
        exit 0
        ;;
    --version)
        if [ -z "${2:-}" ]; then
            error "Version argument required (e.g., --version v1.0.0)"
        fi
        # Override get_latest_version function
        get_latest_version() {
            echo "$2"
        }
        ;;
    --dir)
        if [ -z "${2:-}" ]; then
            error "Directory argument required (e.g., --dir /usr/local/bin)"
        fi
        INSTALL_DIR="$2"
        # Override setup_install_dir function
        setup_install_dir() {
            if [ ! -d "$INSTALL_DIR" ]; then
                mkdir -p "$INSTALL_DIR" || error "Failed to create directory: $INSTALL_DIR"
            fi
            info "Installation directory: $INSTALL_DIR"
        }
        ;;
    *)
        # Use environment variable if set
        if [ -n "${INSTALL_DIR:-}" ]; then
            # Override setup_install_dir function
            setup_install_dir() {
                if [ ! -d "$INSTALL_DIR" ]; then
                    mkdir -p "$INSTALL_DIR" || error "Failed to create directory: $INSTALL_DIR"
                fi
                info "Installation directory: $INSTALL_DIR"
            }
        fi
        ;;
esac

# Run installation
install_ccagents