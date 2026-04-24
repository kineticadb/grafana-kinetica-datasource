# CI Release Failures - Fix Guide

**Date**: 2026-04-17
**Issue**: Release workflow failing with multiple errors

---

## Critical Errors (Must Fix)

### 1. Missing Backend Binaries ❌

**Error:**
```
error: Missing backend binaries in your plugin archive
error: No binary found for `executable` gpx_kinetica_datasource defined in plugin.json
```

**Cause:** Backend binaries in `dist/` are old (from April 7) and not being rebuilt during CI.

**Fix:**
Backend binaries need to be built fresh. The plugin has a Magefile for this.

```bash
# Install mage if not present
go install github.com/magefile/mage@latest

# Build backend binaries for all platforms
~/go/bin/mage -v buildAll

# Or build for specific platform
~/go/bin/mage -v build
```

**Note:** The CI workflow should trigger backend builds automatically, but we need to verify the Magefile.go exists and is correct.

---

### 2. Security Vulnerabilities (Critical) ⚠️

**Critical Severity:**
- `google.golang.org/grpc` - CVE-2026-33186 (CRITICAL)

**High Severity:**
- `go.opentelemetry.io/otel/sdk` - CVE-2026-24051, CVE-2026-39883 (2 CVEs)
- `flatted` (npm) - CVE-2026-32141, CVE-2026-33228
- `serialize-javascript` (npm) - CVE (unnamed)

**Fix for Go Dependencies:**
```bash
# Update vulnerable Go packages
go get -u google.golang.org/grpc@latest
go get -u go.opentelemetry.io/otel/sdk@latest
go mod tidy
```

**Fix for NPM Dependencies:**
```bash
# Update vulnerable npm packages
npm update flatted serialize-javascript
npm audit fix
```

---

### 3. Plugin ID Format Warning ⚠️

**Error:**
```
warning: plugin.json: plugin id should follow the format org-name-type
detail: The plugin ID should be in the format org-name-type (e.g., myorg-myplugin-panel)
```

**Current:** `kinetica-datasource`
**Should be:** `kinetica-kinetica-datasource` (3 parts minimum)

**Fix:** Update `src/plugin.json`:
```json
{
  "id": "kinetica-kinetica-datasource",
  ...
}
```

**Impact:** This is a BREAKING CHANGE - existing installations will need to reinstall with new ID.

---

### 4. Missing Screenshots Warning ⚠️

**Error:**
```
warning: plugin.json: should include screenshots for the Plugin catalog
```

**Fix:** Add screenshots array to `src/plugin.json`:
```json
{
  "info": {
    ...
    "screenshots": [
      {
        "name": "Query Editor",
        "path": "img/screenshots/query-editor.png"
      },
      {
        "name": "Dashboard Example",
        "path": "img/screenshots/dashboard.png"
      }
    ]
  }
}
```

Then add actual screenshot images to `src/img/screenshots/`.

---

### 5. Outdated Grafana Go SDK ⚠️

**Error:**
```
warning: Your Grafana Go SDK is older than 2 months
```

**Current:** `github.com/grafana/grafana-plugin-sdk-go v0.284.0`

**Fix:**
```bash
go get -u github.com/grafana/grafana-plugin-sdk-go
go mod tidy
```

---

## Step-by-Step Fix Process

### Phase 1: Update Dependencies (Security Fixes)

```bash
# 1. Update Go dependencies
go get -u google.golang.org/grpc@latest
go get -u go.opentelemetry.io/otel/sdk@latest
go get -u github.com/grafana/grafana-plugin-sdk-go
go mod tidy

# 2. Update npm dependencies
npm update flatted serialize-javascript
npm audit fix

# 3. Verify updates
go list -m all | grep -E "grpc|otel|grafana-plugin-sdk"
npm ls flatted serialize-javascript
```

### Phase 2: Build Backend Binaries

```bash
# Check if Magefile.go exists
ls -la Magefile.go

# If exists, build all platforms
go run mage.go -l  # List available targets
go run mage.go buildAll

# Verify binaries created
ls -lh dist/gpx_kinetica_datasource_*
```

### Phase 3: Fix Plugin Configuration

**Option A: Keep current ID (less disruptive)**
Add note in documentation that validator warning is expected.

**Option B: Update to proper format (recommended for new plugins)**

1. Update `src/plugin.json`:
```json
{
  "id": "kinetica-kinetica-datasource",
  ...
}
```

2. Update references in:
   - README.md
   - Documentation
   - Installation instructions
   - Any configuration examples

3. Add migration guide for existing users

### Phase 4: Add Screenshots (Optional but Recommended)

```bash
# 1. Create screenshots directory
mkdir -p src/img/screenshots

# 2. Take screenshots of:
#    - Query editor with sample query
#    - Dashboard with Kinetica panels
#    - Configuration page

# 3. Save as PNG files (~800x600 px)
# 4. Update plugin.json with screenshot paths
```

### Phase 5: Test Build Locally

```bash
# Clean previous builds
rm -rf dist/
rm -f *.zip *.sha256

# Build everything
npm run build
go run mage.go buildAll

# Package
npm run package

# Verify archive contains backend binaries
unzip -l kinetica-datasource-1.0.0.zip | grep gpx_kinetica_datasource
```

### Phase 6: Commit and Tag

```bash
# Commit dependency updates
git add go.mod go.sum package.json package-lock.json
git commit -m "fix: update dependencies to resolve security vulnerabilities"

# Commit plugin configuration changes (if updating ID)
git add src/plugin.json
git commit -m "fix: update plugin ID to follow naming convention"

# Push changes
git push

# Create new tag (after all fixes)
git tag v1.0.1
git push origin v1.0.1
```

---

## Expected CI Behavior After Fixes

### What Should Pass:
- ✅ Build completes without errors
- ✅ Backend binaries present for all platforms
- ✅ No critical/high security vulnerabilities
- ✅ Plugin structure validates correctly

### What Will Still Warn (Non-blocking):
- ⚠️ Unsigned plugin (expected for new/unpublished plugins)
- ⚠️ Plugin ID format (if we choose not to change it)
- ⚠️ Screenshots (if not added)

---

## Quick Fix Priority

**MUST FIX (Blocking):**
1. ✅ Security vulnerabilities (go mod tidy + npm audit fix)
2. ✅ Backend binaries (mage buildAll)

**SHOULD FIX (Best Practice):**
3. ⚠️ Plugin ID format
4. ⚠️ Grafana SDK update
5. ⚠️ Screenshots

**CAN IGNORE:**
6. ℹ️ Unsigned plugin (handled during Grafana review)

---

## Troubleshooting

### "Magefile.go not found"

Check if you have a Magefile.go in the repo root. If not, backend builds need different approach.

### "go.mod: invalid Go version"

The go.mod shows `go 1.24.10` which is invalid (Go doesn't have 1.24 yet, current latest is 1.23.x).

**Fix:**
```bash
# Update to valid Go version
sed -i 's/go 1.24.10/go 1.21/' go.mod
go mod tidy
```

### "npm audit fix breaks things"

If `npm audit fix` causes issues:
```bash
# Revert
git checkout package.json package-lock.json

# Manual selective updates
npm update flatted@latest
npm update serialize-javascript@latest
```

---

## Verification Checklist

Before creating release tag:

- [ ] Go dependencies updated (no critical/high CVEs)
- [ ] NPM dependencies updated (no critical/high CVEs)
- [ ] Backend binaries built successfully
- [ ] `go mod tidy` runs without errors
- [ ] `npm run build` completes successfully
- [ ] `npm run package` creates valid archive
- [ ] Archive contains backend binaries
- [ ] All tests pass (`npm run test:ci`, `npm run e2e`)
- [ ] Plugin ID format decided (keep or change)
- [ ] CHANGELOG.md updated
- [ ] Version bumped if needed

---

## Next Steps

1. Start with security fixes (Phase 1)
2. Build backend binaries (Phase 2)
3. Test locally before pushing
4. Create new release tag
5. Monitor CI workflow

---

## References

- [Grafana Plugin Validator](https://github.com/grafana/plugin-validator)
- [Plugin Security](https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin)
- [Backend Plugins](https://grafana.com/developers/plugin-tools/tutorials/build-a-backend-plugin)
