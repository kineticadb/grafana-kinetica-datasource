# node_modules Golang File Fix

## Issue

The Grafana plugin validator detected a Go file in node_modules during CI validation:

```
error: Invalid Go manifest file: node_modules/flatted/golang/pkg/flatted/flatted.go
detail: file node_modules/flatted/golang/pkg/flatted/flatted.go is in the source code but not in the manifest
```

This error blocked the release process and prevented LLM review from running:

```
suspected: LLM review skipped due to errors in go-manifest
detail: Fix the errors reported by go-manifest before LLM review can run.
```

## Root Cause

The `flatted` npm package (version 3.4.2) intentionally includes implementations in multiple programming languages for cross-language JSON serialization:
- **JavaScript** (primary implementation)
- **Go** (`golang/pkg/flatted/flatted.go`)
- **Python** (`python/flatted.py`)
- **PHP** (`php/flatted.php`)

### Why flatted includes Go code:

Flatted is a "super light and fast circular JSON parser" that enables serializing circular/recursive JSON data structures. The Go implementation allows:
1. Cross-language interoperability
2. Serializing in one language (e.g., Node.js) and deserializing in another (e.g., Go)
3. Consistent flatted format across different tech stacks

### Dependency tree:

```
datasource@1.0.0
├── eslint-webpack-plugin@5.0.2
│   └── flatted@3.4.2
└── eslint@9.39.1
    └── file-entry-cache@8.0.0
        └── flat-cache@4.0.1
            └── flatted@3.4.2
```

flatted is a transitive dependency of eslint (dev dependency).

## How the Validator Works

When running validation, the Grafana plugin validator:

1. Scans the source directory for all `.go` files (using `-sourceCodeUri file://./`)
2. Compares found files against `go_plugin_build_manifest` (contains SHA256 hashes of actual backend Go files)
3. Expects ALL `.go` files to be in the manifest

The validator found:
- **Source scan**: Found `node_modules/flatted/golang/pkg/flatted/flatted.go`
- **Manifest**: Only contains `pkg/main.go`, `pkg/plugin/datasource.go`, etc.
- **Result**: ERROR - Go file not in manifest

## Why This Happened

The CI workflow:
1. Checks out code
2. Runs `npm install` → Creates `node_modules/` with flatted
3. Builds plugin with Grafana SDK
4. Runs validator with `-sourceCodeUri file://./`
5. Validator scans entire current directory (including `node_modules/`)
6. Finds `flatted/golang/*.go` and expects it in manifest

## Solution

Added a `postinstall` script to `package.json` that automatically removes the Go code directory after every `npm install`:

**package.json:**
```json
"scripts": {
  "postinstall": "rm -rf node_modules/flatted/golang || true"
}
```

### How it works:

1. **Automatic**: Runs after EVERY `npm install` (local and CI)
2. **Safe**: The `|| true` ensures npm doesn't fail if directory doesn't exist
3. **Targeted**: Only removes the Go files, leaves JavaScript/Python/PHP implementations intact
4. **CI-friendly**: No workflow changes needed - happens automatically

### Why this is the right solution:

✅ **Works with composite actions**: No need to break down `grafana/plugin-actions/build-plugin`
✅ **No workflow modification**: Works with existing CI/CD setup
✅ **Developer-friendly**: Works locally and in CI automatically
✅ **Minimal impact**: Only removes unnecessary Go code
✅ **Future-proof**: Will work even if flatted updates

## Alternative Solutions Considered

### 1. Clean node_modules in CI workflow ❌
```yaml
- name: Clean node_modules Go files
  run: rm -rf node_modules/flatted/golang
```
**Problem**: Can't inject between `npm install` and validation in composite action

### 2. Plugin validator config file ❌
```yaml
# .plugin-validator.yml
excludePaths:
  - "node_modules/**"
```
**Problem**: Validator doesn't support path-level exclusions, only analyzer/rule-level

### 3. Fork grafana/plugin-actions ❌
**Problem**: Maintenance burden, doesn't track upstream updates

### 4. Use .gitignore ❌
**Problem**: Validator scans filesystem, not git-tracked files

### 5. Exclude flatted from dependencies ❌
**Problem**: It's a transitive dependency of eslint (required for linting)

## Verification

### Local verification:
```bash
$ npm install
# postinstall runs automatically

$ find node_modules -name "*.go" | wc -l
0  # ✅ No Go files

$ ls node_modules/flatted/
cjs  esm  index.js  LICENSE  package.json  php  python  README.md  types
# ✅ golang directory removed
```

### CI verification:

The validator will now:
1. Scan source directory
2. Find only `pkg/**/*.go` files (our backend code)
3. Match against `go_plugin_build_manifest`
4. ✅ PASS - all Go files accounted for

## Impact on flatted Functionality

**None**. The JavaScript implementation (which is what we use) remains intact:
- `node_modules/flatted/index.js` ✅
- `node_modules/flatted/cjs/**` ✅
- `node_modules/flatted/esm/**` ✅

The Go/Python/PHP implementations are for developers who want to use flatted in those languages directly. Since we only use flatted via JavaScript (through eslint), removing the Go directory has zero impact on our plugin.

## Files Modified

**package.json:**
```diff
"scripts": {
  "package": "./scripts/package-plugin.sh",
+ "postinstall": "rm -rf node_modules/flatted/golang || true"
},
```

## Related Links

- [flatted npm package](https://www.npmjs.com/package/flatted)
- [flatted GitHub repository](https://github.com/WebReflection/flatted)
- [Grafana Plugin Validator](https://github.com/grafana/plugin-validator)
- [Grafana Plugin Actions](https://github.com/grafana/plugin-actions)

## Timeline

- **2026-04-17**: Discovered "Invalid Go manifest file" error blocking CI release
- **2026-04-17**: Identified flatted as source of Go file in node_modules
- **2026-04-17**: Implemented postinstall script solution
- **2026-04-17**: Verified zero `.go` files in node_modules after fix
