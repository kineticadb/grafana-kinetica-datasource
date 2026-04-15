# Kinetica Datasource Plugin - Validation Guide

This document describes the plugin validation process using the [Grafana Plugin Validator](https://github.com/grafana/plugin-validator), which helps ensure the plugin meets Grafana's publishing requirements.

## Overview

The Grafana Plugin Validator runs 50+ analyzers that check for:
- Security issues (vulnerabilities, virus detection, unsafe code patterns)
- Structural problems (archive format, metadata, binary validation)
- Publishing requirements (license, screenshots, plugin ID format)
- Code quality (broken links, legacy platform usage)

## Prerequisites

- Docker (recommended) or Node.js/Go for local installation
- Built plugin archive (`.zip` file)

## Building the Plugin Archive

### 1. Build Frontend and Backend

```bash
# Install dependencies
npm install

# Build frontend
npm run build

# Build backend (requires Go and mage)
go install github.com/magefile/mage@latest
~/go/bin/mage BuildBackend
```

### 2. Create Distribution Archive

The archive must follow Grafana's naming convention with the plugin ID as the folder name:

```bash
# Create archive directory
mkdir -p kinetica-datasource

# Copy dist contents
cp -r dist/* kinetica-datasource/

# Create zip archive
zip -r kinetica-datasource.zip kinetica-datasource

# Verify archive contents
unzip -l kinetica-datasource.zip
```

Expected archive structure:
```
kinetica-datasource/
├── CHANGELOG.md
├── LICENSE
├── README.md
├── gpx_kinetica_datasource_darwin_amd64
├── gpx_kinetica_datasource_darwin_arm64
├── gpx_kinetica_datasource_linux_amd64
├── gpx_kinetica_datasource_linux_arm64
├── gpx_kinetica_datasource_windows_amd64.exe
├── img/
│   └── logo.svg
├── module.js
├── module.js.map
└── plugin.json
```

## Running the Validator

### Using Docker (Recommended)

Docker includes all security scanning tools (osv-scanner, semgrep, gosec):

```bash
docker run --pull=always -v $(pwd):/plugin grafana/plugin-validator-cli /plugin/kinetica-datasource.zip
```

### Using NPX

```bash
npx -y @grafana/plugin-validator@latest ./kinetica-datasource.zip
```

### Using Local Installation

```bash
git clone git@github.com:grafana/plugin-validator.git
cd plugin-validator/pkg/cmd/plugincheck2
go install
plugincheck2 ./kinetica-datasource.zip
```

## Validation Results

### Current Status (April 2026)

| Severity | Issue | Status |
|----------|-------|--------|
| Recommendation | Sponsorship link | Optional |
| Warning | Unsigned plugin | Expected for new plugins |
| Warning | Missing screenshots | Action needed |
| Warning | Plugin ID format | Requires coordination |

### Issue Details

#### 1. Sponsorship Link (Recommendation)

**Message**: "Consider to add a sponsorship link in your plugin.json file"

**Resolution**: Optional. Add to `src/plugin.json` if desired:
```json
"links": [
  {
    "name": "sponsor",
    "url": "https://kinetica.com/support"
  }
]
```

#### 2. Unsigned Plugin (Warning)

**Message**: "This is a new (unpublished) plugin. This is expected during the initial review process."

**Resolution**: No action needed. Grafana signs plugins after the review process is complete.

#### 3. Missing Screenshots (Warning)

**Message**: "Screenshots are displayed in the Plugin catalog. Please add at least one screenshot."

**Resolution**: Add screenshots to the `src/img/` directory and reference them in `src/plugin.json`:

```json
"screenshots": [
  {
    "name": "Query Editor",
    "path": "img/screenshot-query-editor.png"
  },
  {
    "name": "Configuration",
    "path": "img/screenshot-config.png"
  }
]
```

Recommended screenshot dimensions: 1200x675 pixels (16:9 aspect ratio)

#### 4. Plugin ID Format (Warning)

**Message**: "The plugin ID should be in the format org-name-type (e.g., myorg-myplugin-panel)"

**Current ID**: `kinetica-datasource`
**Recommended**: `kinetica-kinetica-datasource` or `com-kinetica-datasource`

**Resolution**: This is a breaking change that would require:
- Updating `src/plugin.json`
- Updating backend binary naming in `magefile.go`
- Existing users would need to reconfigure their datasources

Consider this change only before initial public release.

## Resolved Issues

### License File Contains Generic Text

**Issue**: The LICENSE file in `dist/` contained Apache 2.0 template placeholders `{yyyy}` and `{name of copyright owner}`.

**Root Cause**: Webpack's filesystem cache was serving a stale LICENSE file.

**Resolution**: Clear the webpack cache and rebuild:
```bash
rm -rf node_modules/.cache
npm run build
```

The correct MIT LICENSE (from project root) is now copied to `dist/`.

## Advanced Validation Options

### Source Code Validation

For additional analyzers that compare the archive against source code:

```bash
docker run -v $(pwd):/plugin grafana/plugin-validator-cli \
  -sourceCodeUri https://github.com/kinetica/grafana-datasource-plugin \
  /plugin/kinetica-datasource.zip
```

### Strict Mode

Returns non-zero exit code for warnings (useful in CI/CD):

```bash
docker run -v $(pwd):/plugin grafana/plugin-validator-cli \
  -strict \
  /plugin/kinetica-datasource.zip
```

### JSON Output

For programmatic processing:

```bash
docker run -v $(pwd):/plugin grafana/plugin-validator-cli \
  -config /path/to/config.yaml \
  /plugin/kinetica-datasource.zip
```

Where `config.yaml` contains:
```yaml
global:
  jsonOutput: true
```

### Debug Mode

Enable verbose logging:

```bash
DEBUG=1 docker run -v $(pwd):/plugin grafana/plugin-validator-cli \
  /plugin/kinetica-datasource.zip
```

## CI/CD Integration

Add validation to your CI pipeline:

```yaml
# .github/workflows/validate.yml
name: Validate Plugin

on:
  push:
    branches: [main]
  pull_request:

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '22'

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install dependencies
        run: npm ci

      - name: Build frontend
        run: npm run build

      - name: Build backend
        run: |
          go install github.com/magefile/mage@latest
          mage BuildBackend

      - name: Create archive
        run: |
          mkdir -p kinetica-datasource
          cp -r dist/* kinetica-datasource/
          zip -r kinetica-datasource.zip kinetica-datasource

      - name: Validate plugin
        run: |
          docker run --rm -v ${{ github.workspace }}:/plugin \
            grafana/plugin-validator-cli \
            /plugin/kinetica-datasource.zip
```

## Security Scanning

The validator uses three open-source security tools:

| Tool | Purpose |
|------|---------|
| [osv-scanner](https://github.com/google/osv-scanner) | Vulnerability detection in dependencies |
| [semgrep](https://semgrep.dev/) | Code pattern analysis for security issues |
| [gosec](https://github.com/securego/gosec) | Go-specific security scanning |

These tools are automatically run when using the Docker image.

## References

- [Grafana Plugin Validator Repository](https://github.com/grafana/plugin-validator)
- [Grafana Plugin Publishing Guide](https://grafana.com/developers/plugin-tools/publish-a-plugin/publish-a-plugin)
- [Plugin.json Schema](https://grafana.com/developers/plugin-tools/reference/plugin-json)
- [Grafana Plugin E2E Testing](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/)
