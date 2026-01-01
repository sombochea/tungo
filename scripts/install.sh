#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_REPO="sombochea/tungo"
INSTALL_DIR="${TUNGO_INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="tungo"

# Utility functions
print_error() {
    echo -e "${RED}✗ Error: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Detect OS and Architecture
detect_platform() {
    local os=$(uname -s)
    local arch=$(uname -m)
    
    case "$os" in
        Linux)
            OS="linux"
            ;;
        Darwin)
            OS="macos"
            ;;
        *)
            print_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
    
    case "$arch" in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
        arm64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    print_info "Detected platform: $OS/$ARCH"
}

# Get latest release version
get_latest_version() {
    print_info "Fetching latest release information..."
    
    local latest_url="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    local version=$(curl -s "$latest_url" | grep '"tag_name"' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/' | head -1)
    
    if [[ -z "$version" ]]; then
        print_error "Could not fetch latest version from GitHub"
        exit 1
    fi
    
    echo "$version"
}

# Download binary
download_binary() {
    local version=$1
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$version/tungo-${OS}-${ARCH}"
    local temp_file=$(mktemp)
    
    print_info "Downloading tungo $version..."
    
    if ! curl -L --progress-bar "$download_url" -o "$temp_file"; then
        print_error "Failed to download binary"
        rm -f "$temp_file"
        exit 1
    fi
    
    echo "$temp_file"
}

# Install binary
install_binary() {
    local temp_file=$1
    
    # Create install directory if it doesn't exist
    if [[ ! -d "$INSTALL_DIR" ]]; then
        print_info "Creating installation directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi
    
    # Move and make executable
    local target_path="$INSTALL_DIR/$BINARY_NAME"
    
    print_info "Installing binary to $target_path..."
    mv "$temp_file" "$target_path"
    chmod +x "$target_path"
    
    print_success "Binary installed successfully"
    echo "$target_path"
}

# Check if directory is in PATH
check_path() {
    local install_dir=$1
    
    if [[ ":$PATH:" == *":$install_dir:"* ]]; then
        return 0
    else
        return 1
    fi
}

# Suggest PATH update
suggest_path_update() {
    local install_dir=$1
    
    if ! check_path "$install_dir"; then
        print_warning "Installation directory is not in your PATH"
        echo ""
        echo "To use 'tungo' command, add the following to your shell profile:"
        echo ""
        
        local shell_name=$(basename "$SHELL")
        case "$shell_name" in
            zsh)
                echo "  echo 'export PATH=\"$install_dir:\$PATH\"' >> ~/.zshrc"
                echo "  source ~/.zshrc"
                ;;
            bash)
                echo "  echo 'export PATH=\"$install_dir:\$PATH\"' >> ~/.bashrc"
                echo "  source ~/.bashrc"
                ;;
            *)
                echo "  export PATH=\"$install_dir:\$PATH\""
                ;;
        esac
        
        echo ""
        print_info "Or run: source <(echo 'export PATH=\"$install_dir:\$PATH\"')"
    fi
}

# Main installation
main() {
    echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║   TunGo Client Installation Script     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════╝${NC}\n"
    
    # Detect platform
    detect_platform
    
    # Get latest version
    VERSION=$(get_latest_version)
    print_info "Latest version: $VERSION"
    
    # Download binary
    TEMP_FILE=$(download_binary "$VERSION")
    
    # Install binary
    INSTALL_PATH=$(install_binary "$TEMP_FILE")
    
    # Verify installation
    print_info "Verifying installation..."
    if ! "$INSTALL_PATH" --version >/dev/null 2>&1; then
        print_warning "Binary verification failed, but installation completed"
    else
        print_success "Installation verified"
    fi
    
    # Print summary
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   Installation Complete!              ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════╝${NC}\n"
    
    echo "Installation Details:"
    echo "  Version: $VERSION"
    echo "  Platform: $OS/$ARCH"
    echo "  Location: $INSTALL_PATH"
    echo ""
    
    # Check PATH
    suggest_path_update "$INSTALL_DIR"
    
    echo ""
    echo "Next steps:"
    echo "  1. Add installation directory to PATH (if needed)"
    echo "  2. Run: tungo --help"
    echo ""
}

# Run main
main "$@"
