# Plugin Signing Guide

## Why Sign Your Plugin?

Plugin signing verifies that your plugin comes from a trusted source and hasn't been tampered with. Grafana validates the signature using a `MANIFEST.txt` file.

### When Signing is Required:

✅ **Required:**
- Publishing to Grafana plugin catalog
- Installing on Grafana Cloud
- Installing on enterprise Grafana instances with security policies

⚠️ **Optional:**
- Private/internal deployments (can bypass with `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS`)
- Development/testing environments

## Step-by-Step Signing Setup

### 1. Create a Grafana Cloud Account

If you don't already have one:

1. Go to https://grafana.com/auth/sign-up
2. Create a free Grafana Cloud account
3. Verify your email address
4. Complete your organization setup

### 2. Generate an Access Policy Token

Access policy tokens have replaced the legacy API keys for plugin signing.

**Navigation Path:**
```
Grafana Cloud → My Account → Security → Access Policies
```

**Or direct link:** https://grafana.com/orgs/YOUR_ORG/access-policies

**Step-by-step:**

1. **Click "Create access policy"**

2. **Configure the policy:**
   - **Name**: `Plugin Signing - kinetica-grafana-datasource`
   - **Realm**: Select your organization (usually `all-stacks`)
   - **Scope**: Select `plugins:write` ✅
     - This grants permission to sign plugins
     - Do NOT select broader scopes for security

3. **Click "Create"**

4. **Generate a token:**
   - Click **"Create token"**
   - **Token name**: `kinetica-grafana-datasource-signing-token`
   - **Expiration**:
     - For production: 1 year (balance between security and maintenance)
     - For testing: 90 days
   - Click **"Create"**

5. **Copy and save the token immediately:**
   ```
   glsa_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   ```

   ⚠️ **IMPORTANT**: This token will only be shown once. Save it securely!

   **Recommended storage:**
   - Password manager (1Password, LastPass, Bitwarden)
   - Encrypted file
   - GitHub Secrets (for CI/CD)

### 3. Add Token to GitHub Secrets

For automated CI/CD signing:

1. **Navigate to your repository**:
   - https://github.com/kineticadb/kinetica-grafana-datasource

2. **Go to Settings → Secrets and variables → Actions**
   - Direct link: https://github.com/kineticadb/kinetica-grafana-datasource/settings/secrets/actions

3. **Click "New repository secret"**

4. **Create the secret:**
   - **Name**: `GRAFANA_ACCESS_POLICY_TOKEN`
   - **Value**: Paste your token (starts with `glsa_`)
   - Click **"Add secret"**

### 4. Verify CI/CD Configuration

The workflows are already configured to use the token:

**`.github/workflows/release.yml`:**
```yaml
- uses: grafana/plugin-actions/build-plugin@build-plugin/v1.0.2
  with:
    go-version: '1.25'
    policy_token: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN }}
```

**`.github/workflows/ci.yml`:**
```yaml
- name: Sign plugin
  run: npm run sign
  if: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN != '' }}
  env:
    GRAFANA_ACCESS_POLICY_TOKEN: ${{ secrets.GRAFANA_ACCESS_POLICY_TOKEN }}
```

## How to Sign Locally (Optional)

For testing or manual releases:

### Method 1: Using npm script

```bash
# Set the environment variable
export GRAFANA_ACCESS_POLICY_TOKEN="glsa_your_token_here"

# Build the plugin first
npm run build
mage buildAll

# Sign it
npm run sign
```

### Method 2: Using npx directly

```bash
# Set the token
export GRAFANA_ACCESS_POLICY_TOKEN="glsa_your_token_here"

# Sign
npx @grafana/sign-plugin@latest
```

### What Happens During Signing

1. **Tool scans dist/ directory**
2. **Generates SHA256 hashes** of all plugin files
3. **Creates MANIFEST.txt** containing:
   - Plugin metadata
   - File hashes
   - Digital signature
4. **Validates plugin.json** structure

**Example MANIFEST.txt:**
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
    "gpx_kinetica_datasource_linux_amd64": "sha256_hash_here",
    ...
  }
}
-----BEGIN PGP SIGNATURE-----
...signature...
-----END PGP SIGNATURE-----
```

## Automated Release Workflow

Once the token is added to GitHub Secrets:

1. **Create a release tag:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions automatically:**
   - ✅ Builds frontend (webpack)
   - ✅ Builds backend (all 6 platforms)
   - ✅ **Signs the plugin** (creates MANIFEST.txt)
   - ✅ Packages the plugin (ZIP with signature)
   - ✅ Generates SHA256 checksum
   - ✅ Creates GitHub release with signed artifact

3. **Verification:**
   - Download the release artifact
   - Unzip it
   - Check for `MANIFEST.txt` file
   - Grafana will validate the signature on installation

## Testing Signed vs Unsigned

### Unsigned Plugin (Development):

```yaml
# docker-compose.yaml
environment:
  - GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=kinetica-grafana-datasource
```

### Signed Plugin (Production):

```yaml
# docker-compose.yaml
environment:
  # No need to allow unsigned plugins
  - GF_LOG_LEVEL=info
```

## Troubleshooting

### Error: "Invalid signature"

**Cause**: Plugin files were modified after signing

**Solution**: Re-sign the plugin
```bash
npm run sign
```

### Error: "Token expired"

**Cause**: Access policy token has expired

**Solution**:
1. Generate a new token in Grafana Cloud
2. Update GitHub Secret `GRAFANA_ACCESS_POLICY_TOKEN`

### Error: "Plugin not signed"

**Cause**: MANIFEST.txt file is missing

**Solution**:
1. Check if `GRAFANA_ACCESS_POLICY_TOKEN` is set in GitHub Secrets
2. Re-run the release workflow
3. Verify the token has `plugins:write` scope

### Warning: "unsigned plugin"

**During development**: Normal - use `GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS`

**During release**: Ensure the workflow ran successfully and created MANIFEST.txt

## Security Best Practices

### Token Security:

✅ **DO:**
- Store token in GitHub Secrets
- Use token expiration (1 year recommended)
- Rotate tokens periodically
- Limit token scope to `plugins:write` only
- Use different tokens for different environments

❌ **DON'T:**
- Commit tokens to git
- Share tokens in plain text
- Use tokens without expiration
- Grant broader scopes than needed
- Reuse tokens across multiple plugins

### Validation:

Before submitting to Grafana catalog:

```bash
# Verify the signature locally
npx @grafana/plugin-validator@latest kinetica-datasource-1.0.0.zip

# Check for:
# - ✅ Valid signature
# - ✅ All files in manifest
# - ✅ Correct plugin metadata
```

## CI/CD Status

Once configured, every release will show in GitHub Actions:

```
✅ Build frontend
✅ Build backend (6 platforms)
✅ Sign plugin (with GRAFANA_ACCESS_POLICY_TOKEN)
✅ Package plugin
✅ Validate signature
✅ Create release
```

## Additional Resources

- [Grafana Plugin Signing Documentation](https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin)
- [Create Access Policies](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/create-access-policies/)
- [Using Access Policy Tokens](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/using-an-access-policy-token/)
- [@grafana/sign-plugin npm package](https://www.npmjs.com/package/@grafana/sign-plugin)

## Quick Start Checklist

- [ ] Create Grafana Cloud account
- [ ] Generate Access Policy token with `plugins:write` scope
- [ ] Add token to GitHub Secrets as `GRAFANA_ACCESS_POLICY_TOKEN`
- [ ] Verify workflows are configured (✅ already done)
- [ ] Create release tag to test signing
- [ ] Verify MANIFEST.txt in release artifact
- [ ] Submit to Grafana plugin catalog (optional)

## Summary

Plugin signing is **enabled in CI/CD** and will automatically sign releases once you:
1. Generate the access policy token in Grafana Cloud
2. Add it as `GRAFANA_ACCESS_POLICY_TOKEN` in GitHub Secrets

No code changes required - the workflows are ready! 🎉
