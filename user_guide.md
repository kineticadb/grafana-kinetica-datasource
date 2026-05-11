# Kinetica Grafana Datasource Plugin

## User Guide

**Version 1.0.1**

*Connecting Grafana to Kinetica*

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [System Requirements](#2-system-requirements)
3. [Installation](#3-installation)
4. [Configuration](#4-configuration)
5. [Usage Guide](#5-usage-guide)
6. [Development Guide](#6-development-guide)
7. [Testing](#7-testing)
8. [Troubleshooting](#8-troubleshooting)
9. [Plugin Distribution](#9-plugin-distribution)
10. [Appendix: Quick Reference](#appendix-a-quick-reference)

---

## 1. Introduction

### 1.1 Overview

The Kinetica Grafana Datasource Plugin enables seamless integration between Grafana and Kinetica, a GPU-accelerated database optimized for real-time analytics. This plugin allows users to query Kinetica databases directly from Grafana dashboards, enabling powerful visualizations of time-series data and complex analytical queries.

Key capabilities of this plugin include:

- Native SQL query support with Kinetica-specific optimizations
- Real-time data streaming for live dashboard updates
- Template variable support for dynamic filtering
- Alerting integration with Grafana's alerting system

### 1.2 Architecture

The plugin consists of two main components that work together to provide a complete data source solution:

| Component | Description |
|-----------|-------------|
| Frontend (React/TypeScript) | Handles the configuration UI, query editor, and visual components within Grafana |
| Backend (Go) | Manages database connections, query execution, and data transformation using the official Kinetica Go API |

This hybrid architecture enables efficient query processing while providing a rich user experience in the Grafana interface.

### 1.3 License

This plugin is released under the Apache License 2.0, allowing free use, modification, and distribution for both commercial and non-commercial purposes.

---

## 2. System Requirements

### 2.1 Software Prerequisites

Before installing the Kinetica Grafana Datasource Plugin, ensure your environment meets the following requirements:

| Component | Minimum Version | Notes |
|-----------|-----------------|-------|
| Grafana | 11.0+ | Enterprise or OSS edition |
| Node.js | 22.0+ | Required for development only |
| Go | 1.24.10+ | Required for backend compilation |
| Docker | 20.10+ | Optional, for containerized deployment |
| npm | 10.9.2+ | Required for frontend build |

### 2.2 Kinetica Database Requirements

The plugin requires connectivity to a Kinetica database instance:

- Kinetica 7.1 or later (recommended: latest stable release)
- Network accessibility from the Grafana server to Kinetica host
- Valid database credentials with appropriate read permissions
- HTTPS enabled for production deployments (recommended)

### 2.3 Network Requirements

Ensure the following network configurations are in place:

- Port 9191 (default Kinetica HTTP port) accessible from Grafana
- Port 8082 (Kinetica HTTP head node) for cluster deployments
- Firewall rules allowing bidirectional communication
- DNS resolution for Kinetica hostnames (if using hostnames instead of IPs)

---

## 3. Installation

### 3.1 Installing from Grafana Plugin Catalog

The recommended method for production environments:

1. Log into your Grafana instance as an administrator
2. Navigate to **Configuration > Plugins**
3. Search for "Kinetica" in the plugin catalog
4. Click on the Kinetica Datasource plugin and select **Install**
5. Restart Grafana to complete the installation

### 3.2 Manual Installation

For environments without internet access or for custom builds:

#### 3.2.1 Download the Plugin

Obtain the plugin distribution archive from the official release page or build from source.

#### 3.2.2 Extract to Plugins Directory

Copy the plugin to your Grafana plugins directory:

```bash
# Default locations:
# Linux: /var/lib/grafana/plugins/
# macOS: /usr/local/var/lib/grafana/plugins/
# Windows: C:\Program Files\GrafanaLabs\grafana\data\plugins\

unzip kinetica-datasource-1.0.0.zip -d /var/lib/grafana/plugins/
```

#### 3.2.3 Configure Unsigned Plugin Loading

For unsigned or development builds, add to `grafana.ini`:

```ini
[plugins]
allow_loading_unsigned_plugins = kinetica-datasource
```

#### 3.2.4 Restart Grafana

```bash
sudo systemctl restart grafana-server
```

### 3.3 Docker Installation

The plugin includes Docker Compose configuration for development and testing:

```bash
# Clone the repository
git clone https://github.com/kinetica/datasource.git
cd datasource

# Start the development environment
docker compose up --build
```

This launches Grafana on port 3000 with the plugin automatically loaded and configured for development.

### 3.4 Verifying Installation

After installation, verify the plugin is correctly loaded:

1. Navigate to **Configuration > Data sources** in Grafana
2. Click "Add data source"
3. Search for "Kinetica" — it should appear in the list
4. Check Grafana logs for any plugin loading errors

---

## 4. Configuration

### 4.1 Adding a Kinetica Data Source

To configure a new Kinetica data source connection:

1. Go to **Configuration > Data sources > Add data source**
2. Select "Kinetica" from the list
3. Enter a descriptive name for this data source
4. Configure the connection settings as described below
5. Click **Save & Test** to verify the connection

### 4.2 Connection Settings

Configure the following settings for your Kinetica instance:

| Setting | Description |
|---------|-------------|
| URL | The full URL to your Kinetica instance (e.g., `http://kinetica.example.com:9191`) |
| User | Database username with query permissions |
| Password | Database password (stored securely in Grafana) |
| Skip TLS Verify | Disable TLS certificate verification (use only for testing/development) |

> **Note:** When running Grafana in Docker and Kinetica on the host machine, use `http://host.docker.internal:9191` as the URL instead of `localhost`.

### 4.3 Advanced Settings

#### 4.3.1 SSL/TLS Configuration

For secure connections:

- Use `https://` protocol in the URL
- Valid SSL certificates should be configured on the Kinetica server
- Keep "Skip TLS Verify" **disabled** in production (default)
- Only enable "Skip TLS Verify" for development/testing with self-signed certificates

#### 4.3.2 Connection Pooling

The backend maintains connection pools for optimal performance. Default pool settings work well for most deployments. For high-concurrency environments, contact Kinetica support for tuning recommendations.

### 4.4 Environment Variables

The plugin supports configuration via environment variables, useful for containerized deployments:

| Variable | Description |
|----------|-------------|
| `KINETICA_HOST` | Kinetica host URL |
| `KINETICA_USER` | Database username |
| `KINETICA_PASSWORD` | Database password |
| `KINETICA_SCHEMA` | Default schema |

Environment variables can be configured in a `.env` file when using Docker Compose.

---

## 5. Usage Guide

### 5.1 Creating Queries

The Kinetica data source supports standard SQL queries with Kinetica-specific extensions.

#### 5.1.1 Basic Query

Enter SQL directly in the query editor:

```sql
SELECT timestamp, value, category
FROM sensor_data
WHERE timestamp >= $__timeFrom
  AND timestamp <= $__timeTo
ORDER BY timestamp ASC
```

#### 5.1.2 Grafana Macros

The plugin supports Grafana's built-in macros for time-range queries:

| Macro | Description |
|-------|-------------|
| `$__timeFrom` | Start of selected time range (epoch milliseconds) |
| `$__timeTo` | End of selected time range (epoch milliseconds) |
| `$__timeFilter(column)` | Generates time filter for specified column |
| `$__interval` | Suggested interval based on panel width |
| `$__timeGroup(column, interval)` | Groups timestamps by specified interval |

### 5.2 Query Editor Features

#### 5.2.1 Code Completion

The query editor provides intelligent code completion for:

- Table and view names from the configured schema
- Column names after selecting a table
- Kinetica-specific SQL functions
- Grafana macros and template variables

#### 5.2.2 Query Formatting

Use the **Format** button to automatically format your SQL for readability. The editor preserves comments and string literals during formatting.

### 5.3 Time Series Data

For time series panels, ensure your query returns data in the expected format:

- First column: timestamp (as epoch or ISO 8601)
- Second column: numeric value
- Additional columns: labels/dimensions (optional)

Example time series query:

```sql
SELECT
  timestamp AS time,
  avg(temperature) AS value,
  location AS metric
FROM weather_stations
WHERE $__timeFilter(timestamp)
GROUP BY $__timeGroup(timestamp, $__interval), location
ORDER BY time ASC
```

### 5.4 Template Variables

Create dynamic dashboards using template variables:

1. Go to **Dashboard Settings > Variables > Add variable**
2. Set the data source to your Kinetica instance
3. Enter a query to populate variable values
4. Use the variable in panel queries as `$variableName`

Example variable query:

```sql
SELECT DISTINCT category
FROM products
ORDER BY category ASC
```

---

## 6. Development Guide

### 6.1 Setting Up the Development Environment

Clone the repository and install dependencies:

```bash
git clone https://github.com/kinetica/datasource.git
cd datasource

# Install frontend dependencies
npm install

# Update Go dependencies
go get -u github.com/grafana/grafana-plugin-sdk-go
go mod tidy
```

### 6.2 Development Commands

The project provides npm scripts for common development tasks:

| Command | Description |
|---------|-------------|
| `npm run dev` | Build frontend in watch mode |
| `npm run build` | Production frontend build |
| `npm run test` | Run unit tests in watch mode |
| `npm run test:ci` | Run all tests (CI mode) |
| `npm run lint` | Run ESLint |
| `npm run lint:fix` | Auto-fix lint issues |
| `npm run server` | Start Docker development server |
| `npm run e2e` | Run Playwright E2E tests |
| `npm run typecheck` | TypeScript type checking |
| `mage -v` | Build Go backend binaries |

### 6.3 Building the Backend

The Go backend uses Mage for build automation:

```bash
# Build all platform binaries
mage -v

# List available build targets
mage -l
```

This generates binaries for Linux, Windows, and macOS in the `dist/` directory.

### 6.4 Running the Development Server

Start a local Grafana instance with the plugin loaded:

```bash
# Start Grafana with Docker Compose
npm run server

# Or with a specific Grafana version
GRAFANA_VERSION=11.3.0 npm run server
```

Access Grafana at `http://localhost:3000` (default credentials: admin/admin).

### 6.5 Project Structure

Key directories in the project:

| Directory | Purpose |
|-----------|---------|
| `src/` | Frontend TypeScript/React source code |
| `pkg/` | Backend Go source code |
| `dist/` | Compiled plugin distribution |
| `provisioning/` | Grafana provisioning configuration |
| `tests/` | E2E Playwright tests |
| `.config/` | Build tool configurations |

---

## 7. Testing

### 7.1 Unit Tests

Run Jest unit tests for the frontend code:

```bash
# Interactive watch mode
npm run test

# CI mode (single run)
npm run test:ci
```

Tests use Jest with the Grafana testing utilities. The timezone is forced to UTC for consistent snapshot testing.

### 7.2 End-to-End Tests

E2E tests use Playwright to verify plugin functionality:

```bash
# Start the test server first
npm run server

# Run E2E tests
npm run e2e
```

Tests run against a local Grafana instance using the `@grafana/plugin-e2e` framework.

### 7.3 Type Checking

Verify TypeScript types without building:

```bash
npm run typecheck
```

### 7.4 Linting

Check code style and potential issues:

```bash
# Check for issues
npm run lint

# Auto-fix issues
npm run lint:fix
```

---

## 8. Troubleshooting

### 8.1 Common Issues

#### 8.1.1 Plugin Not Loading

**Symptoms:** Plugin does not appear in the data source list.

**Solutions:**

- Check Grafana logs for plugin loading errors
- Verify plugin files are in the correct directory
- Ensure `allow_loading_unsigned_plugins` is configured
- Restart Grafana after plugin installation

#### 8.1.2 Connection Errors

**Symptoms:** "Unable to connect to Kinetica" or timeout errors.

**Solutions:**

- Verify the Host URL is accessible from the Grafana server
- Check firewall rules and network connectivity
- Confirm credentials are correct
- Test with curl: `curl -v https://kinetica-host:9191/get/version`

#### 8.1.3 Query Errors

**Symptoms:** Queries return errors or unexpected results.

**Solutions:**

- Test the query directly in Kinetica Workbench
- Check for syntax errors in SQL
- Verify table and column names exist
- Ensure the user has SELECT permissions on queried tables

### 8.2 Enabling Debug Logging

For detailed troubleshooting, enable debug logging in `grafana.ini`:

```ini
[log]
level = debug

[log.filters]
plugin.kinetica-datasource = debug
```

Or via environment variables (Docker):

```bash
GF_LOG_LEVEL=debug
GF_LOG_FILTERS=plugin.kinetica-datasource:debug
```

### 8.3 Docker Networking Issues

When running Grafana in Docker and Kinetica on the host:

- Use `host.docker.internal` instead of `localhost` in the URL field
  - Example: `http://host.docker.internal:9191`
- The `extra_hosts` mapping is already configured in the included docker-compose.yaml
- Ensure Docker's network mode allows host gateway access

> **Important:** Using `localhost` or `127.0.0.1` will not work when Grafana runs in a Docker container, as these addresses refer to the container itself, not the host machine.

### 8.4 Getting Support

For additional assistance:

- Check the GitHub repository issues for known problems
- Review Grafana and Kinetica documentation
- Contact Kinetica support for database-specific issues
- Open a GitHub issue with detailed reproduction steps

---

## 9. Plugin Distribution

### 9.1 Signing the Plugin

For distribution outside development, plugins must be signed:

1. Create a Grafana Cloud account and API key with PluginPublisher role
2. Set the `GRAFANA_API_KEY` secret in your GitHub repository
3. Use the release workflow to create signed builds
4. Or run locally: `npm run sign`

### 9.2 Creating a Release

Use npm version to tag and release:

```bash
# Update version and create git tag
npm version patch  # or minor, major

# Push with tags to trigger release workflow
git push origin main --follow-tags
```

The GitHub Actions workflow will build, sign, and create a release with the plugin archive.

---

## Appendix A: Quick Reference

### A.1 Grafana Macros

| Macro | Example Output |
|-------|----------------|
| `$__timeFrom` | `1704067200000` |
| `$__timeTo` | `1704153600000` |
| `$__timeFilter(ts)` | `ts >= 1704067200000 AND ts <= 1704153600000` |
| `$__interval` | `1m`, `5m`, `1h` (auto-calculated) |
| `$__timeGroup(ts, 1h)` | `FLOOR(ts / 3600000) * 3600000` |

### A.2 Useful npm Commands

| Task | Command |
|------|---------|
| Install dependencies | `npm install` |
| Development build | `npm run dev` |
| Production build | `npm run build` |
| Run tests | `npm run test` |
| Start dev server | `npm run server` |
| Lint code | `npm run lint:fix` |
| E2E tests | `npm run e2e` |

### A.3 Environment Variables

| Variable | Purpose |
|----------|---------|
| `GRAFANA_VERSION` | Specify Grafana version for dev server |
| `GRAFANA_API_KEY` | API key for plugin signing |
| `GF_LOG_LEVEL` | Grafana logging level |
| `GF_LOG_FILTERS` | Plugin-specific log filtering |

### A.4 Key Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `@grafana/data` | 12.3.0+ | Grafana data utilities |
| `@grafana/ui` | 12.3.0+ | Grafana UI components |
| `@grafana/runtime` | 12.3.0+ | Grafana runtime APIs |
| `grafana-plugin-sdk-go` | 0.284.0 | Go backend SDK |
| `kinetica-api-go` | 0.0.5 | Official Kinetica Go client |

---
