# Plugin Signing Workflow Diagram

## Complete Signing Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        PLUGIN SIGNING WORKFLOW                          │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│ STEP 1: Initial Setup (One-time)                                       │
└─────────────────────────────────────────────────────────────────────────┘

    Developer                    Grafana Cloud                 GitHub
       │                               │                          │
       │   1. Create account          │                          │
       ├──────────────────────────────>│                          │
       │   2. Verify email             │                          │
       │<──────────────────────────────│                          │
       │                               │                          │
       │   3. Navigate to              │                          │
       │      Access Policies          │                          │
       ├──────────────────────────────>│                          │
       │                               │                          │
       │   4. Create policy            │                          │
       │      (plugins:write)          │                          │
       ├──────────────────────────────>│                          │
       │                               │                          │
       │   5. Generate token           │                          │
       ├──────────────────────────────>│                          │
       │                               │                          │
       │   6. Receive token            │                          │
       │      glsa_xxxx...             │                          │
       │<──────────────────────────────│                          │
       │                               │                          │
       │   7. Add to GitHub Secrets    │                          │
       │      (GRAFANA_ACCESS_POLICY_TOKEN)                      │
       ├─────────────────────────────────────────────────────────>│
       │                               │                          │

┌─────────────────────────────────────────────────────────────────────────┐
│ STEP 2: Development & Build                                            │
└─────────────────────────────────────────────────────────────────────────┘

    Developer              Build Process             dist/ Output
       │                        │                         │
       │   npm run build        │                         │
       ├───────────────────────>│                         │
       │                        │   ┌─ plugin.json        │
       │                        │   ├─ module.js          │
       │                        ├──>├─ gpx_*_amd64        │
       │   mage buildAll        │   ├─ gpx_*_arm          │
       ├───────────────────────>│   ├─ gpx_*_arm64        │
       │                        │   ├─ gpx_*_darwin_amd64 │
       │                        │   ├─ gpx_*_darwin_arm64 │
       │                        │   ├─ gpx_*_windows.exe  │
       │                        │   ├─ go_plugin_build_manifest
       │                        │   ├─ img/logo.svg       │
       │                        │   ├─ LICENSE            │
       │                        │   ├─ README.md          │
       │                        │   └─ CHANGELOG.md       │
       │                        │                         │

┌─────────────────────────────────────────────────────────────────────────┐
│ STEP 3: Signing Process                                                │
└─────────────────────────────────────────────────────────────────────────┘

    Developer         @grafana/sign-plugin      Grafana API       dist/
       │                     │                      │              │
       │   npm run sign      │                      │              │
       ├────────────────────>│                      │              │
       │                     │                      │              │
       │                     │  1. Read all files   │              │
       │                     ├─────────────────────>│              │
       │                     │                      │              │
       │                     │  2. Calculate SHA256 │              │
       │                     │     for each file    │              │
       │                     │                      │              │
       │                     │  3. Send hashes +    │              │
       │                     │     plugin metadata  │              │
       │                     ├─────────────────────>│              │
       │                     │     + Access Token   │              │
       │                     │                      │              │
       │                     │  4. Verify token     │              │
       │                     │  5. Generate PGP     │              │
       │                     │     signature        │              │
       │                     │                      │              │
       │                     │  6. Return signature │              │
       │                     │<─────────────────────┤              │
       │                     │                      │              │
       │                     │  7. Create           │              │
       │                     │     MANIFEST.txt     │              │
       │                     ├──────────────────────┼─────────────>│
       │                     │                      │  MANIFEST.txt│
       │   ✅ Signed!        │                      │              │
       │<────────────────────┤                      │              │
       │                     │                      │              │

┌─────────────────────────────────────────────────────────────────────────┐
│ STEP 4: CI/CD Automated Signing                                        │
└─────────────────────────────────────────────────────────────────────────┘

    Push Code         GitHub Actions          Signing Service     Release
       │                     │                      │              │
       │   git push          │                      │              │
       ├────────────────────>│                      │              │
       │                     │                      │              │
       │                     │  ┌─ Install deps     │              │
       │                     │  ├─ Run tests        │              │
       │                     │  ├─ Build frontend   │              │
       │                     │  └─ Build backend    │              │
       │                     │                      │              │
       │                     │  Sign plugin step:   │              │
       │                     ├─────────────────────>│              │
       │                     │  Uses:               │              │
       │                     │  GRAFANA_ACCESS_     │              │
       │                     │  POLICY_TOKEN        │              │
       │                     │                      │              │
       │                     │  Creates MANIFEST    │              │
       │                     │<─────────────────────┤              │
       │                     │                      │              │
       │                     │  Package plugin      │              │
       │                     ├──────────────────────┼─────────────>│
       │                     │                      │  Signed ZIP  │
       │                     │                      │              │
       │   ✅ CI Passed      │                      │              │
       │<────────────────────┤                      │              │
       │                     │                      │              │

┌─────────────────────────────────────────────────────────────────────────┐
│ STEP 5: Release Workflow                                               │
└─────────────────────────────────────────────────────────────────────────┘

    Create Tag        GitHub Actions          Build & Sign        Release
       │                     │                      │              │
       │   git tag v1.0.0   │                      │              │
       ├────────────────────>│                      │              │
       │   git push tag      │                      │              │
       │                     │                      │              │
       │                     │  Trigger release     │              │
       │                     │  workflow            │              │
       │                     │                      │              │
       │                     │  ┌─ Build all        │              │
       │                     │  ├─ Run tests        │              │
       │                     │  ├─ Sign plugin      │              │
       │                     │  ├─ Package ZIP      │              │
       │                     │  └─ Validate         │              │
       │                     │                      │              │
       │                     │  Create release      │              │
       │                     ├──────────────────────┼─────────────>│
       │                     │                      │   Artifacts: │
       │                     │                      │   - .zip     │
       │                     │                      │   - .sha256  │
       │   ✅ Release v1.0.0 │                      │              │
       │<────────────────────┴──────────────────────┴──────────────┤
       │                                                            │

┌─────────────────────────────────────────────────────────────────────────┐
│ STEP 6: Grafana Validation                                             │
└─────────────────────────────────────────────────────────────────────────┘

    Install Plugin         Grafana Server       MANIFEST.txt
       │                          │                   │
       │   Load plugin           │                   │
       ├────────────────────────>│                   │
       │                          │                   │
       │                          │  1. Read          │
       │                          │     MANIFEST.txt  │
       │                          ├──────────────────>│
       │                          │                   │
       │                          │  2. Extract       │
       │                          │     PGP signature │
       │                          │     and file list │
       │                          │                   │
       │                          │  3. Verify PGP    │
       │                          │     signature     │
       │                          │     (Grafana key) │
       │                          │                   │
       │                          │  4. Calculate     │
       │                          │     SHA256 of all │
       │                          │     plugin files  │
       │                          │                   │
       │                          │  5. Compare       │
       │                          │     calculated    │
       │                          │     vs manifest   │
       │                          │                   │
       │                          │  ✅ Match!        │
       │   ✅ Plugin loaded       │                   │
       │<─────────────────────────┤                   │
       │   (no unsigned warning)  │                   │
       │                          │                   │
```

## File Structure During Signing

```
BEFORE SIGNING:
dist/
├── plugin.json
├── module.js
├── gpx_kinetica_datasource_linux_amd64
├── gpx_kinetica_datasource_linux_arm
├── gpx_kinetica_datasource_linux_arm64
├── gpx_kinetica_datasource_darwin_amd64
├── gpx_kinetica_datasource_darwin_arm64
├── gpx_kinetica_datasource_windows_amd64.exe
├── go_plugin_build_manifest
├── img/logo.svg
├── LICENSE
├── README.md
└── CHANGELOG.md

AFTER SIGNING:
dist/
├── plugin.json
├── module.js
├── gpx_kinetica_datasource_linux_amd64
├── gpx_kinetica_datasource_linux_arm
├── gpx_kinetica_datasource_linux_arm64
├── gpx_kinetica_datasource_darwin_amd64
├── gpx_kinetica_datasource_darwin_arm64
├── gpx_kinetica_datasource_windows_amd64.exe
├── go_plugin_build_manifest
├── img/logo.svg
├── LICENSE
├── README.md
├── CHANGELOG.md
└── MANIFEST.txt  ← ADDED BY SIGNING
```

## MANIFEST.txt Structure

```
┌─────────────────────────────────────────────────────────────┐
│                      MANIFEST.txt                           │
└─────────────────────────────────────────────────────────────┘

-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA512

┌─────────────────────────────────────────┐
│           Plugin Metadata               │
├─────────────────────────────────────────┤
│ manifestVersion: "2.0.0"                │
│ signatureType: "private"                │
│ signedByOrg: "kinetica"                 │
│ plugin: "kinetica-grafana-datasource"   │
│ version: "1.0.0"                        │
│ keyId: "7e4d0c6a..."                    │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│         File Hash Registry              │
├─────────────────────────────────────────┤
│ "files": {                              │
│   "plugin.json":                        │
│     "sha256:abc123...",                 │
│   "module.js":                          │
│     "sha256:def456...",                 │
│   "gpx_kinetica_datasource_linux_amd64":│
│     "sha256:ghi789...",                 │
│   ... (all files)                       │
│ }                                       │
└─────────────────────────────────────────┘

-----BEGIN PGP SIGNATURE-----

iQIzBAEBCgAdFiEE... (Grafana's signature)
...cryptographic signature...
-----END PGP SIGNATURE-----
```

## Token Scopes Explained

```
┌─────────────────────────────────────────────────────────────────┐
│                     Access Policy Token                         │
└─────────────────────────────────────────────────────────────────┘

Format: glsa_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

┌────────────────────────────────────┐
│         Available Scopes           │
├────────────────────────────────────┤
│                                    │
│  ✅ plugins:write                  │  ← WE USE THIS
│     - Sign plugins                 │
│     - Update plugin metadata       │
│                                    │
│  ❌ plugins:delete                 │  ← NOT NEEDED
│     - Delete plugins               │
│                                    │
│  ❌ org:write                      │  ← TOO BROAD
│     - Modify organization          │
│                                    │
│  ❌ stack:write                    │  ← TOO BROAD
│     - Modify Grafana stacks        │
│                                    │
└────────────────────────────────────┘

Best Practice: Use MINIMUM required scope (plugins:write)
```

## Security Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     Security Model                              │
└─────────────────────────────────────────────────────────────────┘

 Developer                Grafana Cloud           Grafana Server
    │                          │                        │
    │  Token (glsa_xxx)       │                        │
    │  plugins:write only     │                        │
    │                          │                        │
    │  ┌────────────────┐     │                        │
    │  │ Cannot:        │     │                        │
    │  │ - Delete org   │     │                        │
    │  │ - Modify stacks│     │                        │
    │  │ - Access data  │     │                        │
    │  └────────────────┘     │                        │
    │                          │                        │
    │  ┌────────────────┐     │                        │
    │  │ Can only:      │     │                        │
    │  │ - Sign plugins │     │                        │
    │  └────────────────┘     │                        │
    │                          │                        │
    │  Send plugin files +    │                        │
    │  hashes + token          │                        │
    ├─────────────────────────>│                        │
    │                          │                        │
    │                          │  ┌──────────────────┐  │
    │                          │  │ Verify:          │  │
    │                          │  │ 1. Token valid   │  │
    │                          │  │ 2. Scope correct │  │
    │                          │  │ 3. Rate limits   │  │
    │                          │  └──────────────────┘  │
    │                          │                        │
    │                          │  Generate PGP signature│
    │                          │  using Grafana's       │
    │                          │  private key           │
    │                          │                        │
    │  Signed MANIFEST.txt    │                        │
    │<─────────────────────────┤                        │
    │                          │                        │
    │                          │                        │
    │  Upload to Grafana      │                        │
    │  Catalog                 │                        │
    ├─────────────────────────>│                        │
    │                          │                        │
    │                          │  User installs plugin  │
    │                          ├───────────────────────>│
    │                          │                        │
    │                          │                        │  ┌─────────────┐
    │                          │                        │  │ Verify:     │
    │                          │                        │  │ 1. PGP sig  │
    │                          │                        │  │ 2. Hashes   │
    │                          │                        │  │ 3. Metadata │
    │                          │                        │  └─────────────┘
    │                          │                        │
    │                          │  ✅ Plugin validated   │
    │                          │     and loaded         │
    │                          │                        │
```

## Troubleshooting Decision Tree

```
                    npm run sign
                         │
                         ↓
                  Does it succeed?
                    ╱         ╲
                  Yes          No
                   │            │
                   ↓            ↓
            MANIFEST.txt   Check error message
               exists           │
                  │             ├─→ "Invalid token"
                  ↓             │   └─→ Regenerate token in Grafana Cloud
            Package plugin      │       Update GitHub Secret
                  │             │
                  ↓             ├─→ "Token expired"
            Upload to           │   └─→ Create new token (1 year)
             Grafana            │       Update GitHub Secret
                  │             │
                  ↓             ├─→ "Authentication failed"
            ✅ SUCCESS          │   └─→ Verify scope is plugins:write
                                │       Check token starts with glsa_
                                │
                                ├─→ "File not found"
                                │   └─→ Run npm run build
                                │       Run mage buildAll
                                │
                                └─→ "Network error"
                                    └─→ Check internet connection
                                        Try again

                    CI/CD signing fails?
                         │
                         ↓
              Check GitHub Actions logs
                         │
                         ├─→ "Secret not found"
                         │   └─→ Add GRAFANA_ACCESS_POLICY_TOKEN
                         │       to GitHub Secrets
                         │
                         ├─→ "if: false"
                         │   └─→ Secret name must be exactly
                         │       GRAFANA_ACCESS_POLICY_TOKEN
                         │
                         └─→ Step skipped
                             └─→ Check if condition in workflow
                                 Verify secret name spelling
```

## Quick Reference

```
┌────────────────────────────────────────────────────────────────┐
│                      QUICK COMMANDS                            │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Setup (one-time):                                            │
│  1. Get token from Grafana Cloud                              │
│  2. Add to GitHub Secrets                                     │
│                                                                │
│  Local signing:                                               │
│  $ export GRAFANA_ACCESS_POLICY_TOKEN="glsa_xxx"              │
│  $ npm run build && mage buildAll                             │
│  $ npm run sign                                               │
│  $ ls dist/MANIFEST.txt  ← Should exist                       │
│                                                                │
│  Create release:                                              │
│  $ git tag v1.0.0                                             │
│  $ git push origin v1.0.0                                     │
│  → GitHub Actions handles signing automatically               │
│                                                                │
│  Validate:                                                    │
│  $ npx @grafana/plugin-validator@latest plugin.zip            │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

## See Also

- [Signing Quick Reference](SIGNING_QUICK_REFERENCE.md) - 5-minute setup guide
- [Plugin Signing Step-by-Step](PLUGIN_SIGNING_STEP_BY_STEP.md) - Complete walkthrough
- [Grafana Submission Fixes](GRAFANA_SUBMISSION_FIXES.md) - Common issues
