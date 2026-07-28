# Plugin Signing - Quick Reference Card

**5-minute setup guide for signing the kinetica-grafana-datasource plugin**

---

## Prerequisites Checklist
- [ ] Grafana Cloud account created
- [ ] Plugin built (`npm run build && mage buildAll`)
- [ ] GitHub repository admin access

---

## Step 1: Get Your Signing Token (5 minutes)

### Navigate to Access Policies
**Direct URL**: https://grafana.com/orgs/YOUR_ORG/access-policies

Or navigate: Grafana Cloud → My Account → Security → Access Policies

### Create Policy
1. Click **"Create access policy"**
2. **Name**: `Plugin Signing - kinetica-grafana-datasource`
3. **Scope**: ✅ `plugins:write` ONLY
4. Click **"Create"**

### Generate Token
1. Click the policy you just created
2. Click **"Create token"**
3. **Name**: `kinetica-grafana-datasource-signing-token`
4. **Expiration**: 1 year
5. Click **"Create"**
6. **COPY TOKEN IMMEDIATELY** (starts with `glsa_`)

⚠️ **Save this token** - it will only be shown once!

---

## Step 2: Add Token to GitHub (1 minute)

### Navigate to Secrets
**Direct URL**: https://github.com/kineticadb/kinetica-grafana-datasource/settings/secrets/actions

Or navigate: Repository → Settings → Secrets and variables → Actions

### Create Secret
1. Click **"New repository secret"**
2. **Name**: `GRAFANA_ACCESS_POLICY_TOKEN` (exact name!)
3. **Secret**: Paste your `glsa_...` token
4. Click **"Add secret"**

✅ Done! CI/CD will now sign automatically.

---

## Step 3: Test It Works

### Option A: Test in CI (push a version tag)

Signing only runs on a version tag push, not on a branch push — CI does no signing.

```bash
git tag v1.2.3
git push origin v1.2.3
```

**Check**: GitHub Actions → **Release** workflow → the
`Run grafana/plugin-actions/build-plugin@build-plugin/v1.0.2` step should show ✅.
There is no separate "Sign plugin" step; signing happens inside that action.

### Option B: Test Locally (Optional)
```bash
# Set token
export GRAFANA_ACCESS_POLICY_TOKEN="glsa_your_token_here"

# Build plugin
npm run build
mage buildAll

# Sign it
npm run sign

# Verify
ls -lh dist/MANIFEST.txt
```

---

## Step 4: Create Signed Release

```bash
# Create and push tag
git tag v1.0.0
git push origin v1.0.0
```

**Result**: GitHub Actions creates signed release automatically

**Download from**: https://github.com/kineticadb/kinetica-grafana-datasource/releases

---

## Verify Signing Worked

### Check for MANIFEST.txt
```bash
# Download release ZIP
unzip -l kinetica-grafana-datasource-1.0.0.zip | grep MANIFEST.txt
```

Should show:
```
kinetica-grafana-datasource/MANIFEST.txt
```

### Validate Plugin
```bash
npx @grafana/plugin-validator@latest kinetica-grafana-datasource-1.0.0.zip
```

Should show:
```
✅ Valid signature
✅ All files in manifest
```

---

## Troubleshooting

### "Plugin not signed" in CI
**Fix**: Verify GitHub Secret name is exactly `GRAFANA_ACCESS_POLICY_TOKEN`

### "Invalid token"
**Fix**: Generate new token in Grafana Cloud, update GitHub Secret

### "File modified after signing"
**Fix**: Delete `dist/MANIFEST.txt`, rebuild, re-sign

### "unsigned plugin" warning in Grafana
**Fix**: Check `dist/MANIFEST.txt` exists before packaging

---

## What Signing Does

### Before Signing
```
dist/
├── plugin.json
├── module.js
├── gpx_* (binaries)
└── ...
```

### After Signing
```
dist/
├── plugin.json
├── module.js
├── gpx_* (binaries)
├── MANIFEST.txt  ← Added by signing
└── ...
```

### MANIFEST.txt Contains
- SHA256 hash of every file
- PGP signature from Grafana
- Plugin ID and version
- Proves authenticity

---

## Your Workflows (Already Configured ✅)

### .github/workflows/ci.yml

No signing. CI builds, tests, and validates `plugin.json` only.

### .github/workflows/release.yml
```yaml
- uses: grafana/plugin-actions/build-plugin@build-plugin/v1.0.2
  with:
    go-version: '1.26'
    node-version: '22'
    policy_token: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN }}
    attestation: true
```

**No code changes needed!** Just add the token to GitHub Secrets.

---

## Security Reminders

✅ **DO**:
- Store token in GitHub Secrets
- Use 1-year expiration
- Limit scope to `plugins:write`

❌ **DON'T**:
- Commit token to git
- Share token in plain text
- Use tokens without expiration

---

## Quick Commands

```bash
# Local signing test
export GRAFANA_ACCESS_POLICY_TOKEN="glsa_..."
npm run build && mage buildAll && npm run sign

# Create release
git tag v1.0.0 && git push origin v1.0.0

# Validate signed plugin
npx @grafana/plugin-validator@latest plugin.zip

# Check MANIFEST.txt
cat dist/MANIFEST.txt
```

---

## Summary

**Required**: Add `GRAFANA_ACCESS_POLICY_TOKEN` to GitHub Secrets

**That's it!** Everything else is automated.

1. Get token from Grafana Cloud (plugins:write)
2. Add to GitHub Secrets
3. Push code → automatic signing
4. Create tag → automatic signed release

**Your workflows already handle**:
- ✅ Building the plugin
- ✅ Signing with your token
- ✅ Creating MANIFEST.txt
- ✅ Packaging signed plugin
- ✅ Creating GitHub release

---

## Support

**Full guide**: See `docs/PLUGIN_SIGNING_STEP_BY_STEP.md`

**Official docs**: https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin

**Access policies**: https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/
