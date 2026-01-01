#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
FORCE=false
VERSION=""

# Functions
usage() {
    cat << EOF
Usage: ./scripts/release.sh [OPTIONS]

Options:
    -v, --version VERSION    Version to release (e.g., v1.0.0). Required if not using --auto
    -a, --auto              Auto-increment patch version based on latest tag
    -f, --force             Force push and overwrite existing tag
    -h, --help              Show this help message

Examples:
    # Create release with specific version
    ./scripts/release.sh -v v1.0.0

    # Auto-increment patch version and force overwrite
    ./scripts/release.sh --auto --force

    # Create release and force push
    ./scripts/release.sh -v v2.1.0 -f

EOF
    exit 1
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -a|--auto)
            AUTO_INCREMENT=true
            shift
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            ;;
    esac
done

# Get latest tag or use default
get_latest_version() {
    local latest=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    echo "$latest"
}

# Increment version
increment_patch() {
    local version=$1
    # Remove 'v' prefix
    local num_version=${version#v}
    
    # Split version into parts
    IFS='.' read -ra parts <<< "$num_version"
    
    # Increment patch version
    parts[2]=$((${parts[2]:-0} + 1))
    
    # Reconstruct version with 'v' prefix
    echo "v${parts[0]}.${parts[1]}.${parts[2]}"
}

# Validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo -e "${RED}Invalid version format: $version${NC}"
        echo -e "${YELLOW}Expected format: v1.0.0${NC}"
        exit 1
    fi
}

# Check if working directory is clean
check_clean_working_dir() {
    if ! git diff-index --quiet HEAD --; then
        echo -e "${RED}Error: Working directory is not clean${NC}"
        echo -e "${YELLOW}Please commit or stash your changes first${NC}"
        git status
        exit 1
    fi
}

# Main logic
main() {
    echo -e "${BLUE}=== TunGo Release Script ===${NC}\n"
    
    # Check git status
    check_clean_working_dir
    
    # Determine version
    if [[ "$AUTO_INCREMENT" == true ]]; then
        local latest=$(get_latest_version)
        VERSION=$(increment_patch "$latest")
        echo -e "${BLUE}Auto-incrementing from $latest to $VERSION${NC}"
    fi
    
    # Validate version was provided
    if [[ -z "$VERSION" ]]; then
        echo -e "${RED}Error: No version specified${NC}"
        usage
    fi
    
    # Validate version format
    validate_version "$VERSION"
    
    # Check if tag already exists
    if git rev-parse "$VERSION" >/dev/null 2>&1; then
        if [[ "$FORCE" != true ]]; then
            echo -e "${RED}Error: Tag $VERSION already exists${NC}"
            echo -e "${YELLOW}Use -f/--force flag to overwrite${NC}"
            exit 1
        else
            echo -e "${YELLOW}Warning: Tag $VERSION already exists, will force overwrite${NC}"
        fi
    fi
    
    # Create tag
    echo -e "${BLUE}Creating tag: $VERSION${NC}"
    if [[ "$FORCE" == true ]]; then
        git tag -f -a "$VERSION" -m "Release $VERSION"
    else
        git tag -a "$VERSION" -m "Release $VERSION"
    fi
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to create tag${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Tag created: $VERSION${NC}"
    
    # Push tag
    echo -e "${BLUE}Pushing tag to origin...${NC}"
    if [[ "$FORCE" == true ]]; then
        git push --force origin "$VERSION"
    else
        git push origin "$VERSION"
    fi
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to push tag${NC}"
        echo -e "${YELLOW}Rolling back local tag...${NC}"
        git tag -d "$VERSION"
        exit 1
    fi
    
    echo -e "${GREEN}✓ Tag pushed successfully${NC}\n"
    
    # Summary
    echo -e "${GREEN}=== Release Complete ===${NC}"
    echo -e "Version: ${BLUE}$VERSION${NC}"
    echo -e "Status: ${GREEN}Ready for deployment${NC}"
    echo -e "\nWorkflows will start automatically when GitHub processes the tag.\n"
}

# Run main function
main
