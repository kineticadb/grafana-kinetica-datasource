# Plugin Signing - Complete Step-by-Step Guide

## Overview

Plugin signing is the process of digitally signing your plugin package to prove it comes from a trusted source and hasn't been tampered with. Grafana validates the signature using a `MANIFEST.txt` file that contains cryptographic hashes of all plugin files.

## When You Need to Sign

**Required for:**
- ✅ Grafana plugin catalog submission
- ✅ Grafana Cloud installation
- ✅ Enterprise Grafana instances with security policies

**Not required for:**
- Development/testing (use `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS`)
- Private deployments (can bypass signature checking)

## Prerequisites

Before you begin, ensure:
- ✅ Plugin is built (`npm run build` and `mage buildAll` completed)
- ✅ You have a Grafana Cloud account
- ✅ You have admin access to your GitHub repository

---

## Step 1: Create Grafana Cloud Account

If you don't already have a Grafana Cloud account:

### 1.1 Sign Up

1. **Navigate to**: https://grafana.com/auth/sign-up
2. **Fill in**:
   - Email address
   - Password
   - Company name (or personal name)
3. **Click**: "Sign up"
4. **Verify**: Check your email and click the verification link

### 1.2 Complete Organization Setup

1. **Organization name**: Enter your organization name (e.g., "Kinetica")
2. **Organization slug**: This will be used in URLs (e.g., `kinetica`)
3. **Complete** the setup wizard

**Result**: You should now see your Grafana Cloud dashboard

---

## Step 2: Generate Access Policy Token

Access policy tokens are the new way to authenticate with Grafana Cloud APIs (replacing legacy API keys).

### 2.1 Navigate to Access Policies

**Option A - Direct Navigation**:
1. Go to Grafana Cloud homepage
2. Click your profile icon (top right)
3. Select "My Account"
4. Click "Security" in left sidebar
5. Click "Access Policies"

**Option B - Direct URL**:
- https://grafana.com/orgs/YOUR_ORG_NAME/access-policies
- Replace `YOUR_ORG_NAME` with your organization slug

### 2.2 Create New Access Policy

1. **Click**: "Create access policy" button (blue button, top right)

2. **Configure the policy**:
   - **Display name**: `Plugin Signing - kinetica-grafana-datasource`
   - **Realm**: Select `all-stacks` or your specific realm
   - **Scopes**:
     - ✅ Check **`plugins:write`** ONLY
     - ❌ Do NOT check other scopes (security best practice)

3. **Click**: "Create" button

**Result**: You should see the new policy in your list

### 2.3 Generate Token from the Policy

1. **Find**: Your newly created policy in the list
2. **Click**: The policy name to open it
3. **Click**: "Create token" button
4. **Configure token**:
   - **Token name**: `kinetica-grafana-datasource-signing-token`
   - **Expiration**:
     - For production: **1 year** (balance between security and maintenance)
     - For testing: **90 days**
   - **Note**: Optional description
5. **Click**: "Create" button

### 2.4 Copy and Save the Token

**⚠️ CRITICAL**: The token will only be shown ONCE!

1. **Copy**: The entire token (starts with `glsa_`)
2. **Save immediately** to a secure location:
   - Password manager (1Password, LastPass, Bitwarden) ✅ Recommended
   - Encrypted file
   - Secure note application

**Example token format**:
```
glsa_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**DO NOT**:
- ❌ Leave this page without saving the token
- ❌ Share the token in plain text messages
- ❌ Commit the token to git
- ❌ Store in unencrypted files

**Result**: You now have a token that can sign plugins

---

## Step 3: Add Token to GitHub Secrets

This enables automatic signing in your CI/CD workflows.

### 3.1 Navigate to Repository Secrets

1. **Go to**: https://github.com/kineticadb/kinetica-grafana-datasource
2. **Click**: "Settings" tab (top menu bar)
3. **Click**: "Secrets and variables" in left sidebar
4. **Click**: "Actions" (dropdown item)

**Direct URL**: https://github.com/kineticadb/kinetica-grafana-datasource/settings/secrets/actions

### 3.2 Create New Repository Secret

1. **Click**: "New repository secret" button (green button, top right)

2. **Fill in**:
   - **Name**: `GRAFANA_ACCESS_POLICY_TOKEN` (EXACTLY this name)
   - **Secret**: Paste your token (the `glsa_...` string)

3. **Click**: "Add secret" button

**Result**: You should see `GRAFANA_ACCESS_POLICY_TOKEN` in your secrets list

### 3.3 Verify Secret Configuration

The secret should appear as:
```
GRAFANA_ACCESS_POLICY_TOKEN
Updated now by YOUR_USERNAME
```

**⚠️ Important**: The secret name MUST be exactly `GRAFANA_ACCESS_POLICY_TOKEN` because your workflows reference it:
```yaml
policy_token: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN }}
```

---

## Step 4: Verify CI/CD Configuration

Your workflows are already configured to use the token. Let's verify:

### 4.1 Check Release Workflow

```bash
cat .github/workflows/release.yml | grep -A 3 "policy_token"
```

Should show:
```yaml
- uses: grafana/plugin-actions/build-plugin@build-plugin/v1.0.2
  with:
    go-version: '1.25'
    policy_token: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN }}
```

### 4.2 Check CI Workflow

```bash
cat .github/workflows/ci.yml | grep -A 5 "Sign plugin"
```

Should show:
```yaml
- name: Sign plugin
  run: npm run sign
  if: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN != '' }}
  env:
    GRAFANA_ACCESS_POLICY_TOKEN: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN }}
```

**Result**: ✅ Workflows are properly configured

---

## Step 5: Test Signing Locally (Optional)

Before pushing to GitHub, you can test signing on your local machine.

### 5.1 Set Environment Variable

**On Linux/macOS**:
```bash
export GRAFANA_ACCESS_POLICY_TOKEN="glsa_your_token_here"
```

**On Windows PowerShell**:
```powershell
$env:GRAFANA_ACCESS_POLICY_TOKEN = "glsa_your_token_here"
```

**On Windows Command Prompt**:
```cmd
set GRAFANA_ACCESS_POLICY_TOKEN=glsa_your_token_here
```

### 5.2 Ensure Plugin is Built

```bash
# Build frontend
npm run build

# Build backend (all platforms)
mage buildAll
```

**Verify dist/ contents**:
```bash
ls -lh dist/
```

Should show:
- plugin.json
- module.js
- gpx_kinetica_datasource_* (6 binaries)
- go_plugin_build_manifest

### 5.3 Run Signing Command

**Using npm script** (recommended):
```bash
npm run sign
```

**Using npx directly**:
```bash
npx @grafana/sign-plugin@latest
```

### 5.4 Verify MANIFEST.txt Created

```bash
ls -lh dist/MANIFEST.txt
```

**Check contents**:
```bash
cat dist/MANIFEST.txt
```

Should show:
```
-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA512

{
  "manifestVersion": "2.0.0",
  "signatureType": "private",
  "signedByOrg": "kinetica",
  "plugin": "kinetica-grafana-datasource",
  "version": "1.0.0",
  "keyId": "...",
  "files": {
    "module.js": "sha256_hash_here",
    "plugin.json": "sha256_hash_here",
    ...
  }
}
-----BEGIN PGP SIGNATURE-----
...signature...
-----END PGP SIGNATURE-----
```

**Result**: ✅ Plugin is signed locally

---

## Step 6: Test Automated Signing in CI/CD

Now test that signing works automatically in GitHub Actions.

### 6.1 Trigger CI Workflow

**Option A - Push to main branch**:
```bash
git add .
git commit -m "chore: update plugin signing documentation"
git push origin master
```

**Option B - Create test branch and PR**:
```bash
git checkout -b test/signing-verification
git add .
git commit -m "test: verify plugin signing in CI"
git push origin test/signing-verification
# Create PR on GitHub
```

### 6.2 Monitor GitHub Actions

1. **Go to**: https://github.com/kineticadb/kinetica-grafana-datasource/actions
2. **Click**: The running workflow
3. **Watch**: The "Build, lint and unit tests" job

### 6.3 Check Signing Step

Find the "Sign plugin" step:
```
✅ Sign plugin
   npx --yes @grafana/sign-plugin@latest
   ℹ Signing manifest for kinetica-grafana-datasource 1.0.0
   ℹ Signed successfully
```

### 6.4 Verify Artifact Contains MANIFEST.txt

1. **Scroll down**: To "Artifacts" section at bottom of workflow run
2. **Download**: The plugin artifact (e.g., `kinetica-grafana-datasource-1.0.0`)
3. **Extract**: The downloaded ZIP
4. **Check**: `MANIFEST.txt` file exists

**Command to verify**:
```bash
unzip -l kinetica-grafana-datasource-1.0.0.zip | grep MANIFEST.txt
```

**Result**: ✅ Automated signing works in CI

---

## Step 7: Create Signed Release

Now create an official release with a signed plugin package.

### 7.1 Create Version Tag

```bash
# Make sure you're on the main branch with latest changes
git checkout master
git pull origin master

# Create and push tag
git tag v1.0.0
git push origin v1.0.0
```

### 7.2 Monitor Release Workflow

1. **Go to**: https://github.com/kineticadb/kinetica-grafana-datasource/actions
2. **Find**: "Release" workflow (triggered by tag push)
3. **Watch**: The workflow progress

**Workflow steps**:
```
✅ Build frontend
✅ Build backend (6 platforms)
✅ Sign plugin (with GRAFANA_ACCESS_POLICY_TOKEN)
✅ Package plugin
✅ Validate plugin
✅ Create GitHub release
✅ Upload signed artifact
```

### 7.3 Verify Release Artifact

1. **Go to**: https://github.com/kineticadb/kinetica-grafana-datasource/releases
2. **Click**: The v1.0.0 release
3. **Download**: `kinetica-grafana-datasource-1.0.0.zip`
4. **Extract**: The ZIP file

**Verify MANIFEST.txt**:
```bash
unzip kinetica-grafana-datasource-1.0.0.zip
ls kinetica-grafana-datasource/MANIFEST.txt
cat kinetica-grafana-datasource/MANIFEST.txt
```

**Result**: ✅ Official signed release created

---

## Step 8: Validate Signed Plugin

Use Grafana's official validator to verify the signature.

### 8.1 Using Plugin Validator (Recommended)

```bash
npx @grafana/plugin-validator@latest kinetica-grafana-datasource-1.0.0.zip
```

**Expected output**:
```
✅ Valid signature
✅ All files in manifest
✅ Plugin metadata valid
✅ Backend detected (6 platforms)
```

### 8.2 Using Docker Validator

```bash
docker run --pull=always \
  -v $PWD/kinetica-grafana-datasource-1.0.0.zip:/archive.zip \
  grafana/plugin-validator-cli /archive.zip
```

### 8.3 Test in Grafana

**Without allowing unsigned plugins**:
```yaml
# docker-compose.yaml - REMOVE unsigned plugin allowance
environment:
  - GF_LOG_LEVEL=info
  # NOT needed for signed plugin:
  # - GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=kinetica-grafana-datasource
```

**Start Grafana**:
```bash
# Extract plugin
unzip kinetica-grafana-datasource-1.0.0.zip

# Copy to Grafana plugins directory
docker compose up -d

# Check logs - should NOT show unsigned plugin warning
docker logs kinetica-datasource 2>&1 | grep -i "unsigned\|signature"
```

**Expected**: No unsigned plugin warnings ✅

**Result**: ✅ Signed plugin works in Grafana

---

## What Happens During Signing

### Input Files (from dist/)
- plugin.json
- module.js
- gpx_kinetica_datasource_* (6 binaries)
- go_plugin_build_manifest
- img/logo.svg
- LICENSE
- README.md
- CHANGELOG.md

### Signing Process
1. **@grafana/sign-plugin tool**:
   - Scans all files in dist/
   - Generates SHA256 hash for each file
   - Sends hashes to Grafana Cloud signing API
   - Receives PGP signature from Grafana

2. **Creates MANIFEST.txt**:
   ```
   -----BEGIN PGP SIGNED MESSAGE-----
   {
     "plugin": "kinetica-grafana-datasource",
     "version": "1.0.0",
     "files": {
       "file1": "sha256_hash",
       "file2": "sha256_hash",
       ...
     }
   }
   -----BEGIN PGP SIGNATURE-----
   [Grafana's cryptographic signature]
   -----END PGP SIGNATURE-----
   ```

3. **Saves to dist/MANIFEST.txt**

### Verification by Grafana
When the plugin loads:
1. Grafana reads MANIFEST.txt
2. Verifies PGP signature against Grafana's public key
3. Recalculates SHA256 hashes of all files
4. Compares calculated hashes with manifest hashes
5. **If match**: ✅ Plugin loads
6. **If mismatch**: ❌ Plugin rejected

---

## Troubleshooting

### Error: "Invalid token" or "Authentication failed"

**Cause**: Token is incorrect, expired, or has wrong permissions

**Solution**:
1. Go back to Grafana Cloud Access Policies
2. Verify the token hasn't expired
3. Regenerate a new token if needed
4. Ensure scope is `plugins:write`
5. Update GitHub Secret with new token

### Error: "Plugin not signed" in CI

**Cause**: GitHub Secret not configured or wrong name

**Solution**:
1. Check secret name is exactly `GRAFANA_ACCESS_POLICY_TOKEN`
2. Verify secret exists in repository settings
3. Re-add the secret if needed
4. Re-run workflow

### Error: "File modified after signing"

**Cause**: Plugin files were changed after MANIFEST.txt was created

**Solution**:
```bash
# Remove old manifest
rm dist/MANIFEST.txt

# Rebuild plugin
npm run build
mage buildAll

# Re-sign
npm run sign
```

### Warning: "unsigned plugin" in Grafana logs

**Cause**: MANIFEST.txt is missing from the plugin package

**Solution**:
1. Verify dist/MANIFEST.txt exists before packaging
2. Ensure package script includes MANIFEST.txt
3. Recreate package:
   ```bash
   npm run sign
   npm run package
   ```

### Token Expired

**Symptoms**: Signing fails with "token expired" error

**Solution**:
1. Go to Grafana Cloud → Access Policies
2. Find your policy
3. Click "Create token" to generate new token
4. Update GitHub Secret `GRAFANA_ACCESS_POLICY_TOKEN`
5. Re-run workflow

---

## Security Best Practices

### Token Management

✅ **DO**:
- Store token in GitHub Secrets
- Use 1-year expiration (balance security/maintenance)
- Limit scope to `plugins:write` only
- Rotate tokens periodically
- Use different tokens for different environments
- Revoke tokens immediately if compromised

❌ **DON'T**:
- Commit tokens to git
- Share tokens in messages/emails
- Use tokens without expiration
- Grant broader scopes than needed
- Log tokens in CI output
- Reuse tokens across multiple plugins

### Environment Variables

**For local testing only**:
```bash
# Add to .bashrc or .zshrc temporarily
export GRAFANA_ACCESS_POLICY_TOKEN="glsa_..."

# Unset after use
unset GRAFANA_ACCESS_POLICY_TOKEN
```

**Never add to**:
- ❌ .env files (tracked by git)
- ❌ Shell history (use space before command)
- ❌ Docker images
- ❌ Public CI logs

---

## Signing Checklist

Use this checklist when setting up plugin signing:

### Initial Setup
- [ ] Create Grafana Cloud account
- [ ] Navigate to Access Policies
- [ ] Create policy with `plugins:write` scope
- [ ] Generate token (1-year expiration)
- [ ] Save token securely (password manager)
- [ ] Add token to GitHub Secrets as `GRAFANA_ACCESS_POLICY_TOKEN`
- [ ] Verify workflows reference the secret correctly

### Local Testing (Optional)
- [ ] Export GRAFANA_ACCESS_POLICY_TOKEN locally
- [ ] Build plugin (`npm run build && mage buildAll`)
- [ ] Run `npm run sign`
- [ ] Verify dist/MANIFEST.txt created
- [ ] Check MANIFEST.txt contains all files

### CI/CD Testing
- [ ] Push commit to trigger CI workflow
- [ ] Verify "Sign plugin" step succeeds
- [ ] Download artifact and check MANIFEST.txt
- [ ] Create version tag (e.g., v1.0.0)
- [ ] Verify release workflow succeeds
- [ ] Download release artifact
- [ ] Extract and verify MANIFEST.txt in release

### Validation
- [ ] Run `npx @grafana/plugin-validator@latest` on package
- [ ] Verify all checks pass
- [ ] Test plugin in Grafana without unsigned allowance
- [ ] Check logs for no signature warnings

### Grafana Submission
- [ ] Upload signed package to Grafana.com
- [ ] Wait for Grafana Labs review
- [ ] Address any feedback
- [ ] Monitor for approval

---

## Summary

**Plugin signing flow**:
```
1. Create Grafana Cloud account
   ↓
2. Generate Access Policy token (plugins:write scope)
   ↓
3. Add token to GitHub Secrets (GRAFANA_ACCESS_POLICY_TOKEN)
   ↓
4. CI/CD automatically signs on every build
   ↓
5. MANIFEST.txt created with file hashes + signature
   ↓
6. Package includes signed MANIFEST.txt
   ↓
7. Submit signed package to Grafana
   ↓
8. Grafana validates signature
   ↓
9. Plugin approved and published
```

**Key points**:
- Token starts with `glsa_`
- Token scope must be `plugins:write`
- GitHub Secret name must be `GRAFANA_ACCESS_POLICY_TOKEN`
- MANIFEST.txt proves plugin authenticity
- Signing happens automatically in CI/CD
- Signed plugins work without `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS`

**Your workflows are ready!** Just add the token to GitHub Secrets and signing will happen automatically on every build and release.

---

## Additional Resources

- [Official Grafana Plugin Signing Docs](https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin)
- [Access Policy Documentation](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/)
- [@grafana/sign-plugin npm package](https://www.npmjs.com/package/@grafana/sign-plugin)
- [Plugin Validator](https://grafana.com/developers/plugin-tools/publish-a-plugin/validate-a-plugin)

---

**Need help?** If you encounter issues:
1. Check the troubleshooting section above
2. Review GitHub Actions logs for error messages
3. Verify token permissions in Grafana Cloud
4. Consult the official Grafana documentation
