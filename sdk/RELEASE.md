# SDK Release Guide

## 🚀 Quick Release

```bash
# Interactive release
./scripts/quick-release.sh

# Direct command
./scripts/release-sdk.sh both 1.0.1
./scripts/release-sdk.sh python 1.0.2
./scripts/release-sdk.sh node 1.0.3
```

Push to trigger release:
```bash
git push origin main
git push origin sdk-1.0.1
```

## 🔧 Setup (First Time)

Add secrets in GitHub **Settings → Secrets and variables → Actions**:

| Secret | Get From | Format |
|--------|----------|--------|
| `PYPI_API_TOKEN` | https://pypi.org/manage/account/token/ | `pypi-AgEI...` |
| `NPM_TOKEN` | https://www.npmjs.com/settings/tokens | `npm_xxx...` |

## 🏷️ Tag Patterns

| Tag | Releases | Example |
|-----|----------|---------|
| `sdk-*` | Both SDKs | `sdk-1.0.0` |
| `sdk-python-*` | Python only | `sdk-python-1.0.1` |
| `sdk-node-*` | Node.js only | `sdk-node-1.0.2` |

## 📋 Release Checklist

### Before Release
- [ ] Python tests pass: `cd sdk/python && uv run pytest tests/`
- [ ] Node.js tests pass: `cd sdk/node && npm test`
- [ ] Git working directory is clean
- [ ] On main branch with latest changes

### Release
- [ ] Run: `./scripts/quick-release.sh`
- [ ] Review: `git log -1` and `git show <tag>`
- [ ] Push: `git push origin main && git push origin <tag>`
- [ ] Monitor GitHub Actions workflow

### After Release
- [ ] Verify on PyPI: https://pypi.org/project/tungo-sdk/
- [ ] Verify on npm: https://www.npmjs.com/package/@tungo/sdk
- [ ] Test install: `pip install tungo-sdk==<version>`
- [ ] Test install: `npm install @tungo/sdk@<version>`

## 🔄 Automated Workflow

When you push a tag matching `sdk-*`:

**Python SDK** (`sdk-python-*` or `sdk-*`):
1. Updates version in `pyproject.toml`
2. Builds with `uv build`
3. Publishes to PyPI
4. Creates GitHub release

**Node.js SDK** (`sdk-node-*` or `sdk-*`):
1. Updates version in `package.json`
2. Runs tests
3. Builds with `npm run build`
4. Publishes to npm with provenance
5. Creates GitHub release

## 📦 Version Examples

```bash
# Patch (bug fixes)
./scripts/release-sdk.sh both 1.0.1

# Minor (new features)
./scripts/release-sdk.sh both 1.1.0

# Major (breaking changes)
./scripts/release-sdk.sh both 2.0.0

# Pre-release
./scripts/release-sdk.sh both 1.1.0-beta.1
```

## 🛠️ Troubleshooting

**Authentication errors:**
- Check secrets are configured correctly
- PyPI/npm tokens may be expired

**Version already exists:**
- PyPI doesn't allow re-upload, increment version
- npm: use `npm unpublish` (within 72h) or increment

**Build fails:**
- Test locally: `uv build` (Python) or `npm run build` (Node.js)
- Check GitHub Actions logs

## ⏪ Rollback

**Before pushing:**
```bash
git tag -d sdk-1.0.1
git reset --hard HEAD~1
```

**After release:**
```bash
# Deprecate npm package
npm deprecate @tungo/sdk@1.0.1 "Please upgrade to 1.0.2"

# Release fix immediately
./scripts/release-sdk.sh both 1.0.2
```

## 📚 Manual Process

If scripts don't work:

```bash
# 1. Update versions
cd sdk/python && sed -i '' 's/version = .*/version = "1.0.1"/' pyproject.toml
cd sdk/node && npm version 1.0.1 --no-git-tag-version

# 2. Commit and tag
git add sdk/python/pyproject.toml sdk/node/package.json
git commit -m "Release SDK v1.0.1"
git tag -a sdk-1.0.1 -m "Release SDK v1.0.1"

# 3. Push
git push origin main
git push origin sdk-1.0.1
```
