# Packaging and Release Guide

This document explains how to package the Grafana Kinetica datasource plugin for distribution and create GitHub releases.

---

## Quick Start

### Manual Packaging

```bash
# 1. Build the plugin
npm run build

# 2. Package the plugin (creates zip + SHA256 hash)
npm run package

# Output:
# - kinetica-datasource-<version>.zip
# - kinetica-datasource-<version>.zip.sha256
```

### Automated Release (via GitHub)

```bash
# 1. Ensure all changes are committed
git add .
git commit -m "Release v1.0.0"
git push

# 2. Create and push a version tag
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions will automatically:
# - Build the plugin
# - Run tests
# - Create release artifacts
# - Generate SHA256 hash
```

---

## Packaging Details

### What Gets Packaged

The `npm run package` script creates a deployment archive from the `dist/` directory containing:

**Frontend Assets:**
- `module.js` - Main plugin JavaScript bundle
- `plugin.json` - Plugin metadata
- `README.md` - Plugin documentation
- `CHANGELOG.md` - Version history
- `LICENSE` - Apache 2.0 license
- `img/logo.svg` - Plugin logo

**Backend Binaries** (if present):
- `gpx_kinetica_datasource_linux_amd64`
- `gpx_kinetica_datasource_linux_arm64`
- `gpx_kinetica_datasource_darwin_amd64`
- `gpx_kinetica_datasource_darwin_arm64`
- `gpx_kinetica_datasource_windows_amd64.exe`

**Note**: Source maps (`*.map` files) are excluded from the package to reduce size.

### Archive Naming Convention

```
kinetica-datasource-<version>.zip
kinetica-datasource-<version>.zip.sha256
```

**Example:**
```
kinetica-datasource-1.0.0.zip
kinetica-datasource-1.0.0.zip.sha256
```

Version is read from `package.json` or can be specified:
```bash
./scripts/package-plugin.sh 1.0.0
```

---

## SHA256 Hash

The packaging script automatically generates a SHA256 hash file for integrity verification.

**Hash file format:**
```
07d548c5ddeb7f9f64c699dccf749140df0f88c4f3f15ef52e74b23f4e6f0d56  ./kinetica-datasource-1.0.0.zip
```

**Verification** (for users installing the plugin):
```bash
# Linux
sha256sum -c kinetica-datasource-1.0.0.zip.sha256

# macOS
shasum -a 256 -c kinetica-datasource-1.0.0.zip.sha256
```

---

## Manual Release Process

### 1. Prepare Release

```bash
# Update version in package.json
npm version 1.0.0 --no-git-tag-version

# Update CHANGELOG.md with release notes
vim CHANGELOG.md

# Commit changes
git add package.json CHANGELOG.md
git commit -m "Prepare release v1.0.0"
git push
```

### 2. Build and Package

```bash
# Clean previous builds
rm -rf dist/

# Build production version
npm run build

# Package for distribution
npm run package
```

**Output:**
```
✓ Archive created: ./kinetica-datasource-1.0.0.zip
✓ Hash file created: ./kinetica-datasource-1.0.0.zip.sha256

Archive size: 93M
SHA256: 07d548c5ddeb7f9f64c699dccf749140df0f88c4f3f15ef52e74b23f4e6f0d56
```

### 3. Test the Package Locally

```bash
# Extract to Grafana plugins directory
unzip kinetica-datasource-1.0.0.zip -d /var/lib/grafana/plugins/kinetica-datasource

# Restart Grafana
sudo systemctl restart grafana-server

# Verify plugin loads
tail -f /var/log/grafana/grafana.log | grep kinetica
```

### 4. Create Git Tag

```bash
# Create annotated tag
git tag -a v1.0.0 -m "Release version 1.0.0"

# Push tag to GitHub
git push origin v1.0.0
```

### 5. Create GitHub Release

**Option A: Via GitHub UI**

1. Go to https://github.com/kineticadb/kinetica-grafana-datasource/releases/new
2. Select tag: `v1.0.0`
3. Release title: `Kinetica Datasource Plugin v1.0.0`
4. Description: Copy from CHANGELOG.md
5. Upload files:
   - `kinetica-datasource-1.0.0.zip`
   - `kinetica-datasource-1.0.0.zip.sha256`
6. Click "Publish release"

**Option B: Via GitHub CLI**

```bash
gh release create v1.0.0 \
  kinetica-datasource-1.0.0.zip \
  kinetica-datasource-1.0.0.zip.sha256 \
  --title "Kinetica Datasource Plugin v1.0.0" \
  --notes-file CHANGELOG.md
```

---

## Automated Release Process (Recommended)

The repository includes a GitHub Actions workflow (`.github/workflows/release.yml`) that automates the release process.

### Workflow Trigger

The workflow runs automatically when you push a version tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

### What the Workflow Does

1. **Checkout code** - Gets the repository at the tagged version
2. **Build plugin** - Uses `grafana/plugin-actions/build-plugin@v1.0.2`
   - Runs `npm install`
   - Runs `npm run build`
   - Creates plugin archive
   - Generates SHA256 hash
3. **Create GitHub release** - Attaches artifacts automatically

### Workflow Configuration

**File:** `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - 'v*' # Triggers on version tags like v1.0.0

jobs:
  release:
    permissions:
      contents: write
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: grafana/plugin-actions/build-plugin@build-plugin/v1.0.2
```

**Note:** Plugin signing is currently commented out. Uncomment the `policy_token` section when ready to sign plugins for Grafana Cloud.

---

## Plugin Signing (Optional)

For publishing to Grafana's official plugin catalog, plugins must be signed.

### Setup Signing

1. **Generate access policy token** at https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin#generate-an-access-policy-token

2. **Add token to repository secrets:**
   - Go to GitHub repository → Settings → Secrets and variables → Actions
   - Add secret: `GRAFANA_ACCESS_POLICY_TOKEN`

3. **Update release workflow** (`.github/workflows/release.yml`):

```yaml
- uses: grafana/plugin-actions/build-plugin@build-plugin/v1.0.2
  with:
    policy_token: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN }}
```

4. **Sign manually** (for testing):

```bash
npm run build
npm run sign
```

This creates a `MANIFEST.txt` file with signature information.

---

## Version Management

### Semantic Versioning

This plugin follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (1.x.x): Breaking changes
- **MINOR** (x.1.x): New features, backward compatible
- **PATCH** (x.x.1): Bug fixes, backward compatible

### Updating Version

**In package.json:**
```json
{
  "version": "1.0.0"
}
```

**Using npm:**
```bash
npm version patch  # 1.0.0 → 1.0.1
npm version minor  # 1.0.1 → 1.1.0
npm version major  # 1.1.0 → 2.0.0
```

**Note:** Use `--no-git-tag-version` if you want to manage git tags separately.

---

## Troubleshooting

### "dist directory not found"

**Problem:** Running `npm run package` before building

**Solution:**
```bash
npm run build
npm run package
```

### "Archive size is too large"

**Problem:** Backend binaries make the archive ~93MB

**Expected:** This is normal for plugins with backend binaries for multiple platforms.

**To reduce size:** Remove unnecessary backend binaries for platforms you don't support.

### "Hash verification failed"

**Problem:** Archive was modified after hash generation

**Solution:**
```bash
# Regenerate hash
./scripts/package-plugin.sh
```

### "Permission denied" when running package script

**Problem:** Script not executable

**Solution:**
```bash
chmod +x scripts/package-plugin.sh
```

---

## Distribution Checklist

Before creating a release, ensure:

- [ ] All tests pass (`npm run test:ci`, `npm run e2e`)
- [ ] Code is linted (`npm run lint`)
- [ ] Type checking passes (`npm run typecheck`)
- [ ] CHANGELOG.md is updated
- [ ] Version bumped in package.json
- [ ] All changes committed to git
- [ ] Plugin builds successfully (`npm run build`)
- [ ] Package creates without errors (`npm run package`)
- [ ] Tested package locally in Grafana
- [ ] Git tag created and pushed
- [ ] Release notes prepared

---

## Files Reference

### Packaging Scripts

- **`scripts/package-plugin.sh`** - Main packaging script
- **`package.json`** - Contains `"package"` npm script

### GitHub Workflows

- **`.github/workflows/release.yml`** - Automated release workflow
- **`.github/workflows/ci.yml`** - CI tests (runs on all commits)

### Plugin Configuration

- **`src/plugin.json`** - Plugin metadata (ID, version placeholders)
- **`dist/plugin.json`** - Built plugin metadata (version filled in)
- **`package.json`** - NPM package configuration

---

## Additional Resources

- [Grafana Plugin Development](https://grafana.com/developers/plugin-tools/)
- [Grafana Plugin Actions](https://github.com/grafana/plugin-actions)
- [Sign a Plugin](https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin)
- [Publish a Plugin](https://grafana.com/developers/plugin-tools/publish-a-plugin)
- [Semantic Versioning](https://semver.org/)
