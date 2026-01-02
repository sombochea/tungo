#!/usr/bin/env bash

# TunGo SDK Release Script
# Automates version bumping and tag creation for SDK releases
#
# Usage:
#   ./scripts/release-sdk.sh [python|node|both] [version] [-f|--force]
#   ./scripts/release-sdk.sh python 1.0.1
#   ./scripts/release-sdk.sh node 1.0.1
#   ./scripts/release-sdk.sh both 1.0.1
#   ./scripts/release-sdk.sh both 1.0.1 --force

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Parse arguments
SDK_TYPE="${1:-both}"
VERSION="${2}"
FORCE_FLAG=""
FORCE_PUSH=""

# Check for force flag
if [[ "${3}" == "-f" || "${3}" == "--force" ]]; then
    FORCE_FLAG="-f"
    FORCE_PUSH="--force"
    echo -e "${YELLOW}Force mode enabled - will overwrite existing tags${NC}"
fi

# Validate arguments
if [[ ! "$SDK_TYPE" =~ ^(python|node|both)$ ]]; then
    echo -e "${RED}Error: Invalid SDK type. Must be 'python', 'node', or 'both'${NC}"
    echo "Usage: $0 [python|node|both] [version]"
    exit 1
fi

if [[ -z "$VERSION" ]]; then
    echo -e "${RED}Error: Version is required${NC}"
    echo "Usage: $0 [python|node|both] [version]"
    echo "Example: $0 both 1.0.1"
    exit 1
fi

# Validate version format (semantic versioning)
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    echo -e "${RED}Error: Invalid version format. Must follow semantic versioning (e.g., 1.0.0 or 1.0.0-beta.1)${NC}"
    exit 1
fi

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     TunGo SDK Release Script          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}SDK Type:${NC} $SDK_TYPE"
echo -e "${GREEN}Version:${NC}  $VERSION"
echo ""

# Check if git working directory is clean
if [[ -n $(git status -s) ]]; then
    echo -e "${YELLOW}Warning: You have uncommitted changes${NC}"
    git status -s
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Release cancelled${NC}"
        exit 1
    fi
fi

# Check if on main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    echo -e "${YELLOW}Warning: You are not on the main branch (current: $CURRENT_BRANCH)${NC}"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Release cancelled${NC}"
        exit 1
    fi
fi

# Function to update Python version
update_python_version() {
    echo -e "${BLUE}Updating Python SDK version...${NC}"
    cd "$PROJECT_ROOT/sdk/python"
    
    # Update pyproject.toml
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s/^version = .*/version = \"$VERSION\"/" pyproject.toml
    else
        # Linux
        sed -i "s/^version = .*/version = \"$VERSION\"/" pyproject.toml
    fi
    
    echo -e "${GREEN}✓ Updated pyproject.toml${NC}"
    
    # Show the change
    grep "^version" pyproject.toml
}

# Function to update Node.js version
update_node_version() {
    echo -e "${BLUE}Updating Node.js SDK version...${NC}"
    cd "$PROJECT_ROOT/sdk/node"
    
    # Update package.json using npm
    npm version "$VERSION" --no-git-tag-version --allow-same-version
    
    echo -e "${GREEN}✓ Updated package.json${NC}"
    
    # Show the change
    cat package.json | grep '"version"'
}

# Update versions based on SDK type
if [[ "$SDK_TYPE" == "python" || "$SDK_TYPE" == "both" ]]; then
    update_python_version
fi

if [[ "$SDK_TYPE" == "node" || "$SDK_TYPE" == "both" ]]; then
    update_node_version
fi

# Commit changes
echo ""
echo -e "${BLUE}Committing version changes...${NC}"
cd "$PROJECT_ROOT"

if [[ "$SDK_TYPE" == "both" ]]; then
    git add sdk/python/pyproject.toml sdk/node/package.json sdk/node/package-lock.json
    COMMIT_MSG="Release SDK v$VERSION"
    TAG_NAME="sdk-$VERSION"
elif [[ "$SDK_TYPE" == "python" ]]; then
    git add sdk/python/pyproject.toml
    COMMIT_MSG="Release Python SDK v$VERSION"
    TAG_NAME="sdk-python-$VERSION"
else
    git add sdk/node/package.json sdk/node/package-lock.json
    COMMIT_MSG="Release Node.js SDK v$VERSION"
    TAG_NAME="sdk-node-$VERSION"
fi

git commit -m "$COMMIT_MSG" || echo -e "${YELLOW}No changes to commit${NC}"

# Create and push tag
echo ""
if [[ -n "$FORCE_FLAG" ]]; then
    echo -e "${YELLOW}Creating/overwriting tag: $TAG_NAME${NC}"
else
    echo -e "${BLUE}Creating tag: $TAG_NAME${NC}"
fi
git tag $FORCE_FLAG -a "$TAG_NAME" -m "$COMMIT_MSG"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Release prepared successfully!    ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review the changes:"
echo "   git log -1"
echo "   git show $TAG_NAME"
echo ""
echo "2. Push to trigger the release:"
echo -e "   ${GREEN}git push origin $CURRENT_BRANCH${NC}"
if [[ -n "$FORCE_PUSH" ]]; then
    echo -e "   ${GREEN}git push origin $TAG_NAME $FORCE_PUSH${NC}"
else
    echo -e "   ${GREEN}git push origin $TAG_NAME${NC}"
fi
echo ""
echo "3. The GitHub Actions workflow will:"
if [[ "$SDK_TYPE" == "python" || "$SDK_TYPE" == "both" ]]; then
    echo "   - Build and publish Python SDK to PyPI"
fi
if [[ "$SDK_TYPE" == "node" || "$SDK_TYPE" == "both" ]]; then
    echo "   - Build and publish Node.js SDK to npm"
fi
echo "   - Create a GitHub release with artifacts"
echo ""
echo -e "${YELLOW}To undo this release (before pushing):${NC}"
echo "   git reset --hard HEAD~1"
echo "   git tag -d $TAG_NAME"
echo ""
