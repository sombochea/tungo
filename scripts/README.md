# Release Scripts

Automated scripts for releasing TunGo SDK packages to PyPI and npm.

## Quick Start

### Interactive Release (Recommended)

```bash
./scripts/quick-release.sh
```

This will:
1. Ask which SDK to release (both, Python, or Node.js)
2. Show current versions
3. Prompt for new version
4. Confirm before proceeding
5. Execute the release

### Command Line Release

```bash
# Release both SDKs with version 1.0.1
./scripts/release-sdk.sh both 1.0.1

# Release only Python SDK
./scripts/release-sdk.sh python 1.0.2

# Release only Node.js SDK
./scripts/release-sdk.sh node 1.0.3
```

## What the Scripts Do

1. **Validate** version format (semantic versioning)
2. **Update** version in `pyproject.toml` and/or `package.json`
3. **Commit** version changes to git
4. **Create** annotated git tag with appropriate name
5. **Provide** instructions for pushing to trigger CI/CD

## Tag Naming Convention

The tag name determines what gets released:

- `sdk-X.Y.Z` → Both Python and Node.js SDKs
- `sdk-python-X.Y.Z` → Python SDK only
- `sdk-node-X.Y.Z` → Node.js SDK only

## After Running the Script

The script prepares everything but **does not push automatically**. You must manually push:

```bash
# Push the commit
git push origin main

# Push the tag (this triggers the release)
git push origin sdk-1.0.1
```

## Undo Before Pushing

If you made a mistake before pushing:

```bash
# Remove the tag
git tag -d sdk-1.0.1

# Undo the commit
git reset --hard HEAD~1
```

## GitHub Actions Workflow

Once you push the tag, GitHub Actions (`.github/workflows/release-sdk.yaml`) will:

### For Python SDK
1. Build the package
2. Run checks with twine
3. Publish to PyPI
4. Create GitHub release with artifacts

### For Node.js SDK
1. Install dependencies
2. Run tests
3. Build the package
4. Publish to npm
5. Create GitHub release with artifacts

## Required GitHub Secrets

Configure in **Settings → Secrets and variables → Actions**:

- `PYPI_API_TOKEN` - PyPI API token for publishing
- `NPM_TOKEN` - npm automation token for publishing

## Version Format

Follow [Semantic Versioning](https://semver.org/):

- `X.Y.Z` - Standard release (e.g., `1.0.0`)
- `X.Y.Z-prerelease` - Pre-release (e.g., `1.0.0-beta.1`)

**Guidelines:**
- **Major (X)**: Breaking changes
- **Minor (Y)**: New features, backward compatible
- **Patch (Z)**: Bug fixes, backward compatible

## Examples

```bash
# Patch release for bug fixes
./scripts/release-sdk.sh both 1.0.1

# Minor release with new features
./scripts/release-sdk.sh both 1.1.0

# Major release with breaking changes
./scripts/release-sdk.sh both 2.0.0

# Beta pre-release
./scripts/release-sdk.sh both 1.1.0-beta.1

# Release only updated SDK
./scripts/release-sdk.sh python 1.0.2  # Python had a bug fix
./scripts/release-sdk.sh node 1.0.3     # Node.js had a bug fix
```

## Troubleshooting

### "Error: Invalid version format"
- Use semantic versioning: `X.Y.Z` or `X.Y.Z-prerelease`
- Valid: `1.0.0`, `2.1.3`, `1.0.0-beta.1`
- Invalid: `v1.0.0`, `1.0`, `1.0.0.0`

### "Warning: You have uncommitted changes"
- Commit or stash your changes first
- Or answer 'y' to continue anyway (not recommended)

### "Warning: You are not on the main branch"
- Checkout main branch: `git checkout main`
- Or answer 'y' to continue from current branch

### GitHub Actions fails
- Check workflow logs in GitHub Actions tab
- Verify secrets are configured correctly
- Ensure tests pass locally before releasing

## See Also

- [SDK Release Guide](../sdk/RELEASE.md) - Comprehensive release documentation
- [Testing Guide](../sdk/TESTING.md) - How to test SDKs before release
- [GitHub Workflow](../.github/workflows/release-sdk.yaml) - CI/CD configuration
