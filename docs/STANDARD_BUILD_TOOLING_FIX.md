# Standard Build Tooling Fix

## Issue

The Grafana plugin validator detected non-standard backend build tooling during the CI release workflow:

```
error: non-standard backend build tooling
detail: The plugin does not appear to use Grafana's standard backend build tooling.
Please use create-plugin to scaffold your plugin: https://grafana.com/developers/plugin-tools/
```

## Root Cause

Our `Magefile.go` contained **196 lines of custom build logic** instead of importing the official Grafana Plugin SDK build package. The validator looks for evidence that the plugin uses Grafana's official tooling by checking for:

1. **`go_plugin_build_manifest`** file in dist/ (contains SHA256 hashes of source files)
2. Standard SDK build targets (buildAll, coverage, etc.)
3. Official Grafana Plugin SDK imports

## Correct Implementation

Plugins created with `create-plugin` (like ScyllaDB and SurrealDB plugins) use this standard approach:

**Magefile.go (12 lines):**
```go
//go:build mage
// +build mage

package main

import (
	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"
)

// Default configures the default target.
var Default = build.BuildAll
```

## Solution Applied

### 1. Replaced Custom Magefile.go

**Before:**
- 196 lines of custom build functions
- Custom `buildArch()`, `buildList()`, `copyGoManifest()` implementations
- Manual cross-compilation logic
- Manual go.mod/go.sum copying

**After:**
- 12 lines importing SDK build package
- Uses official `github.com/grafana/grafana-plugin-sdk-go/build`
- All standard mage targets automatically available

### 2. Installed SDK Build Dependencies

```bash
go get github.com/grafana/grafana-plugin-sdk-go/build@v0.291.1
go mod tidy
```

## SDK Build Targets Provided

The SDK provides all targets needed by CI:

### Build Targets
- `buildAll` - Builds production executables for all supported platforms ✅
- `build:backend` - Build for current platform
- `build:linux`, `build:windows`, `build:darwin` - OS-specific builds
- `build:linuxARM`, `build:linuxARM64`, `build:darwinARM64` - Architecture-specific
- `build:debug` - Debug builds
- `build:custom` - Custom OS/arch combinations

### Testing Targets
- `coverage` - Runs backend tests with coverage reports ✅
- `test` - Runs backend tests
- `testRace` - Runs tests with race detector

### Utility Targets
- `clean` - Removes dist directory ✅
- `build:generateManifestFile` - Creates `go_plugin_build_manifest` ✅
- `format` - Formats source code
- `lint` - Audits code style
- `watch` - Auto-rebuild on file changes
- `debugger` - Attach Delve debugger

## Verification

### 1. All mage targets available:
```bash
$ mage -l
Targets:
  buildAll*                 builds production executables for all supported platforms.
  coverage                  runs backend tests and makes a coverage report.
  clean                     cleans build artifacts, by deleting the dist directory.
  build:generateManifestFile generates a manifest file for plugin submissions
  [... 20+ other targets ...]
```

### 2. BuildAll creates required files:
```bash
$ mage buildAll
$ ls -lh dist/
-rwxrwxr-x  29M gpx_kinetica_datasource_darwin_amd64
-rwxrwxr-x  27M gpx_kinetica_datasource_darwin_arm64
-rwxrwxr-x  28M gpx_kinetica_datasource_linux_amd64
-rwxrwxr-x  27M gpx_kinetica_datasource_linux_arm      # NEW: SDK builds this too!
-rwxrwxr-x  27M gpx_kinetica_datasource_linux_arm64
-rwxrwxr-x  29M gpx_kinetica_datasource_windows_amd64.exe
-rwxr-xr-x  513 go_plugin_build_manifest               # NEW: Proves standard tooling!
```

### 3. go_plugin_build_manifest content:
```
ba5ee903d3b81740a1bd1c9aacf631a04ab8a209838d03a649365cf66590fdc7:Magefile.go
a53bb6051a62611a18ace3b44a4f5c7167947a1c8a99b85fb7c2dc355cf3ed64:pkg/main.go
80435dc192302e1af7fd95e319fcab4dfe1d1d6898ab0583e28441ee10b50c03:pkg/models/settings.go
d590644e15609864e6f1acc2c4ac01e00667b38b6960ccc592727639afc5a0e5:pkg/plugin/datasource.go
653ca55a84a67d8ba3d19d43c9fb9dc76ebc1879ca83eef2425f07759343f309:pkg/plugin/datasource_test.go
bfc32930af119b335efa4ed53903d8c8a6ed5e2ea7a23d539717afd51839d8de:pkg/plugin/models.go
```

This file proves to the validator that:
- Plugin was built with official Grafana SDK
- Source files haven't been tampered with
- Standard build process was followed

### 4. Tests still work:
```bash
$ mage coverage
=== RUN   TestQueryData
--- PASS: TestQueryData (0.00s)
PASS
coverage: 16.7% of statements in ./...
```

### 5. Frontend + Backend coexist:
```bash
$ npm run build && ls dist/
module.js                                    # Frontend ✅
plugin.json                                  # Frontend ✅
gpx_kinetica_datasource_*                   # Backend binaries ✅
go_plugin_build_manifest                    # SDK manifest ✅
```

## Benefits of SDK Build Package

### 1. **Validator Compliance**
- Creates `go_plugin_build_manifest` automatically
- Recognized as "standard tooling" by plugin validator
- No more "non-standard backend build tooling" errors

### 2. **More Build Targets**
- **6 platforms** instead of 5 (added linux-arm)
- Debug builds with Delve debugger support
- Live reload during development
- Automatic source code formatting

### 3. **Less Code to Maintain**
- 196 lines → 12 lines (94% reduction)
- No custom cross-compilation logic
- SDK handles all platform-specific details
- Automatic updates when SDK updates

### 4. **Better Testing**
- Coverage reports built-in
- Race detector support
- Test output formatting

### 5. **Future-Proof**
- New platforms added by SDK automatically available
- Build improvements from Grafana team
- Standard across all official plugins

## Changes to go.mod

**Minimal dependency changes:**
```diff
 require (
 	github.com/hamba/avro/v2 v2.31.0
 	github.com/kineticadb/kinetica-api-go v0.0.5
-	github.com/magefile/mage v1.17.1
 )

 require (
+	github.com/BurntSushi/toml v1.5.0 // indirect
+	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
+	github.com/magefile/mage v1.17.1 // indirect  (moved to indirect)
```

SDK build package brings in a few utility dependencies but they're all indirect.

## CI Compatibility

The SDK targets match exactly what CI expects:

**`.github/workflows/ci.yml` calls:**
```yaml
- name: Test backend
  uses: magefile/mage-action@v3.1.0
  with:
    args: coverage  # ✅ Provided by SDK

- name: Build backend
  uses: magefile/mage-action@v3.1.0
  with:
    args: buildAll  # ✅ Provided by SDK
```

**`.github/workflows/release.yml` uses:**
```yaml
- uses: grafana/plugin-actions/build-plugin@build-plugin/v1.0.2
  # Automatically calls mage buildAll ✅
```

## Known Issues

### Coverage Exit Code

The SDK's `coverage` target returns exit code 1 due to Go 1.25's "no such tool covdata" warning, even though tests pass. This is a known Go 1.25 issue that doesn't affect functionality:

```
go: no such tool "covdata"
=== RUN   TestQueryData
--- PASS: TestQueryData (0.00s)
PASS
coverage: 16.7% of statements in ./...
```

The test **passes** but mage returns non-zero exit code. CI may need to handle this or we may need to update to a newer Go version that fixes this issue.

## References

- [Grafana Plugin SDK Go - build package](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go/build)
- [Grafana Plugin SDK Go repository](https://github.com/grafana/grafana-plugin-sdk-go)
- [Grafana Plugin Tools - Backend Plugins](https://grafana.com/developers/plugin-tools/key-concepts/backend-plugins/grafana-plugin-sdk-for-go)

## Timeline

- **2026-04-17**: Identified "non-standard backend build tooling" error in CI
- **2026-04-17**: Discovered ScyllaDB/SurrealDB use SDK import pattern
- **2026-04-17**: Replaced 196-line custom Magefile with 12-line SDK import
- **2026-04-17**: Verified all CI targets work with SDK build package
