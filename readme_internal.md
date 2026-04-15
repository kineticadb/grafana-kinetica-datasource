# Internal Development Guide

## Files to Check In

When committing code changes, include the following:

### Always Check In

- `src/**` — Frontend TypeScript/React source files
- `pkg/**` — Backend Go source files
- `go.mod`, `go.sum` — Go dependencies (if changed)
- `package.json`, `package-lock.json` — npm dependencies (if changed)
- `CHANGELOG.md` — Update with your changes
- `provisioning/**` — Grafana provisioning configs (if changed)

### Never Check In

- `node_modules/`
- `dist/`
- `artifacts/`
- `coverage/`
- `.env` (contains secrets)

## Docker Build Steps

### 1. Install Dependencies

```bash
npm install
```

### 2. Build Frontend

```bash
# Development (watch mode)
npm run dev

# Production
npm run build
```

### 3. Build Backend

```bash
mage -v
```

### 4. Start Docker Environment

```bash
npm run server
```

Grafana runs at `http://localhost:3000` (admin/admin).

### 5. Lint Before Commit

```bash
npm run lint:fix
npm run typecheck
```

## Quick Reference

| Task | Command |
|------|---------|
| Start dev server | `npm run server` |
| Build all | `npm run build && mage -v` |
| Lint & fix | `npm run lint:fix` |
| Specific Grafana version | `GRAFANA_VERSION=11.3.0 npm run server` |

## Running with Existing Grafana Installation

### 1. Build the Plugin

```bash
# Frontend
npm install
npm run build

# Backend (generates binaries for all platforms)
mage -v
```

### 2. Copy to Grafana Plugins Directory

```bash
# Linux
cp -r dist/ /var/lib/grafana/plugins/kinetica-datasource

# macOS
cp -r dist/ /usr/local/var/lib/grafana/plugins/kinetica-datasource

# Windows
xcopy dist\ "C:\Program Files\GrafanaLabs\grafana\data\plugins\kinetica-datasource" /E /I
```

### 3. Allow Unsigned Plugin Loading

Add to `grafana.ini`:

```ini
[plugins]
allow_loading_unsigned_plugins = kinetica-datasource
```

### 4. Restart Grafana

```bash
# Linux (systemd)
sudo systemctl restart grafana-server

# macOS (Homebrew)
brew services restart grafana

# Windows
net stop Grafana && net start Grafana
```

### 5. Verify

Go to **Configuration → Data sources → Add data source** and search for "Kinetica".