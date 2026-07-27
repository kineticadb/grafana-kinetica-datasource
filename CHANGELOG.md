# Changelog

All notable changes to the Kinetica Grafana Datasource Plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [1.0.8] - 2026-07-27 - https://github.com/kineticadb/grafana-kinetica-datasource/releases/tag/v1.0.8

### Added
- Backend tests for the concurrency pattern used by `QueryData`: result collection,
  error isolation, and mutex-guarded map writes verified under `-race`. These
  exercise a standalone replica of the pattern rather than driving `QueryData`
  itself with multiple queries.
- `golang.org/x/sync` dependency (provides `errgroup`).

### Changed
- Multiple queries in a single request now execute concurrently using `errgroup`
  instead of sequentially, so a panel with several queries is no longer bounded by
  the sum of its query times. A single-query request keeps the direct path and skips
  the goroutine overhead. Per-query failures stay isolated — one failing query does
  not cancel the others.
- The metadata helpers `getSchemas()`, `getTableNames()`, and `getColumns()` now
  return `ResourceFetchResult<string[]>` (`{ data, error }`) rather than a bare
  `string[]`. This is a breaking change for any code importing the `DataSource`
  class directly.
- Query editor styling moved from inline `style` props to Emotion `css` classes that
  use theme spacing tokens, so spacing follows the active Grafana theme instead of
  hard-coded pixel values.
- Updated license to MIT in `package.json`.

### Fixed
- Single quotes in query-builder filter values are now escaped. Previously a value
  such as `O'Brien` terminated the string literal and produced malformed SQL.
- Metadata fetch failures are now surfaced in the query editor as a dismissible
  "Connection Error" alert. Previously they were logged to the browser console and
  the schema, table, and column dropdowns simply rendered empty, giving no
  indication that the datasource was unreachable.
- Replaced an `any`-typed parameter object in `getColumns` with `Record<string, string>`.


## [1.0.7] - 2026-06-17 - https://github.com/kineticadb/grafana-kinetica-datasource/releases/tag/v1.0.7

Initial release of the Kinetica Grafana Datasource Plugin. Earlier `v1.0.0`–`v1.0.6`
tags exist in the repository but were internal pre-release iterations, so this entry
documents the plugin as first shipped.

### Requirements
- Grafana 12.3.0 or later. The plugin is built against Grafana SDK 12.3.0 and
  declares `grafanaDependency` to match, so compatibility with older Grafana is not
  claimed or tested. See `docs/DEPENDENCY_MISMATCH_ANALYSIS.md` for the technical
  rationale.
- Node.js 22 or later and Go 1.26.x to build from source.

### Added
- Visual SQL Query Builder with support for:
  - Schema and table selection
  - Column selection with aggregation functions (AVG, COUNT, MAX, MIN, SUM, STDDEV, VAR)
  - JOIN operations (INNER, LEFT, RIGHT, FULL) with multiple conditions
  - WHERE clause filters with AND/OR logic
  - GROUP BY with HAVING clause support
  - ORDER BY with ASC/DESC sorting
  - LIMIT and OFFSET pagination
  - Set operations (UNION, UNION ALL, INTERSECT, EXCEPT)
- Raw SQL mode with Monaco code editor
- Schema, table, and column autocomplete backed by live Kinetica metadata calls
- Time series support with automatic time column detection
- Time range macros: `$__timeFilter(col)`, `$__timeFrom()`, `$__timeTo()`,
  `$__unixEpochFrom()`, `$__unixEpochTo()`
- Backend plugin with Go SDK integration
- Health check endpoint for connection validation
- Secure credential storage using Grafana's encrypted `secureJsonData`
- Provisioning support for datasource and dashboards
- Docker Compose development environment
- E2E test suite with Playwright
- Documentation: test environment setup, plugin validation, publishing compliance
  report, dependency mismatch analysis, create-plugin tool analysis, and work logs
  under `docs/work-logs/`

### Known limitations
- The shipped E2E suite covers 7 stable scenarios (alerts, provisioning, basic page
  loads), narrowed from an initial 34 so the suite passes reliably across supported
  Grafana versions. Broader coverage is limited by cross-version testing constraints;
  see `E2E_TESTS_README.md` for the testing strategy.
