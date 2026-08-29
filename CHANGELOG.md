# Changelog

All notable changes to the Kinetica Grafana Datasource Plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file documents the versions published to the
[Grafana plugin catalog](https://grafana.com/grafana/plugins/kinetica-grafana-datasource/).
Intermediate tags and GitHub releases used during development are not listed, so the
version numbers here are not contiguous.


## [1.0.12] - <TBD> - https://github.com/kineticadb/grafana-kinetica-datasource/releases/tag/v1.0.12

### Added
- Dashboard variables of type *Query* can now be populated from Kinetica. Select the
  Kinetica datasource on the variable and enter SQL returning one row per value.
  - A single result column supplies both the label and the value.
  - `__text` / `__value` columns display one thing and store another, matching the
    convention used by Grafana's other SQL datasources.
  - Backend macros work in a variable query, and one variable query may reference
    another (chained variables).
- `$__timeGroup(column, interval)` macro, for bucketing a time column in a `GROUP BY`.
  Panels render the bucketed column as a time series.
  - Accepts `500ms`, `30s`, `5m`, `1h`, `7d`, `2w`, quoted or bare, a plain number of
    milliseconds, and Grafana's `$__interval`.
- The query builder's Schema, Table and Column dropdowns now work when Schema or Table
  is set to a dashboard variable.
  - Changing the variable's value re-populates the dropdowns.
  - Dashboard variables are offered as choices in the Schema, Table and Join Table
    dropdowns, grouped above the fetched names and labelled with what each currently
    resolves to (`$schema - currently ki_catalog`). Selecting one stores the variable,
    not its current value.
  - Those dropdowns also accept a typed-in name.
  - A multi-value variable uses its first value.
  - A variable with no value is reported as such instead of an unexplained failure.
- Kinetica `date` and `datetime` columns are now plotted as time series; previously only
  `timestamp` columns were, and the others arrived as text.

### Fixed
- A join against a table in another schema now generates the correct SQL. The base
  table's schema was applied to it instead, producing an invalid table name.

### Changed
- Transitive dependencies for security issues:
  - `react-use` 17.6.0 -> 17.6.1, which moved `js-cookie` 2.2.1 -> 3.0.8 (GHSA-qjx8-664m-686j).
  - `immutable` 5.1.4 -> 5.1.9 (GHSA-v56q-mh7h-f735, GHSA-wf6x-7x77-mvgw, GHSA-xvcm-6775-5m9r).


## [1.0.11] - 2026-08-24 - https://github.com/kineticadb/grafana-kinetica-datasource/releases/tag/v1.0.11

### Changed
- Transitive dependencies for security issues:
  - `fast-uri` 3.1.4 -> 3.1.6 for (GHSA-7p8r-x3mc-p8w7 / CVE-2026-18446)
  - `js-yaml` 4.3.0 -> 4.3.1 and 3.15.0 -> 3.15.1 (GHSA-5p4m-2wfm-xmqj / CVE-2026-59870)
  - `nanoid` 3.3.16 -> 3.3.18 (GHSA-2v37-7h3g-55p8 / CVE-2026-67213)
  - `brace-expansion` 1.1.16 -> 1.1.18, 2.1.2 -> 2.1.4 and 5.0.8 -> 5.0.9
    (GHSA-mh99-v99m-4gvg, GHSA-rgw5-rvv9-x895)


## [1.0.10] - 2026-08-24 - https://github.com/kineticadb/grafana-kinetica-datasource/releases/tag/v1.0.10

### Added
- Dashboard and template variables are now interpolated into queries via
  `applyTemplateVariables`, in both raw SQL and query-builder modes. Single-value
  variables interpolate verbatim so they can be used as identifiers
  (`SELECT * FROM $table`); multi-value and *Include All* variables are quoted and
  comma-joined so they work in an `IN` list. Single quotes in values are escaped,
  reusing `escapeStringValue` from `sqlGenerator.ts` (now exported). Grafana's
  built-in `$__from` / `$__to` / `$__interval` become usable as a side effect.
  The builder's `schema` and `table` fields are interpolated too, so the backend's
  time-column metadata lookup keeps working when the table name is dynamic.
- First jest unit tests (`src/datasource.test.ts`) covering the interpolation
  formatter and `applyTemplateVariables`.
- README documents the template-variable behavior, including the need to use
  `${var:sqlstring}` for single-value string values (a bare `'$var'` is not escaped,
  matching how Grafana's own SQL datasources behave).

### Known limitations
- Dashboard variables still cannot be *defined* by a Kinetica query: the datasource
  does not implement `metricFindQuery`, so a *Query*-type dashboard variable cannot
  be populated from Kinetica.


## [1.0.9] - 2026-07-28 - https://github.com/kineticadb/grafana-kinetica-datasource/releases/tag/v1.0.9

### Added
- `.npmrc` sets `engine-strict=true`, so `npm install` fails immediately when the
  active Node version does not satisfy the `engines` field in `package.json`.
  Previously npm only warned, and the mismatch surfaced later as a confusing runtime
  error from a dependency (for example `Array.prototype.toSorted` being undefined on
  Node 18).

### Changed
- `github.com/grafana/grafana-plugin-sdk-go` -> v0.294.0 (from v0.292.1).
- **Minimum Go toolchain for building from source is now 1.26.5** (was 1.26.3). The
  `go` directive in `go.mod` was raised because the Grafana Go SDK from v0.293.0
  onwards declares `go 1.26.5`. CI is unaffected — the workflows pin
  `go-version: '1.26'`, which resolves to the latest 1.26.x.

### Fixed
- The release workflow now pins `node-version: '22'` for
  `grafana/plugin-actions/build-plugin`. The action defaults to Node 20, so release
  builds were running on a Node version that `package.json` `engines` declares
  unsupported (`>=22`), previously surfacing only as an `EBADENGINE` warning. With
  `engine-strict=true` now in `.npmrc`, that warning would have become a hard
  `npm install` failure in the release build.
- Security issues (Go)
  - `GO-2026-5841`:  `github.com/klauspost/compress` -> v1.19.0
    (fixed as of v1.18.7; resolved to v1.19.0 by the SDK upgrade above)
  - `GO-2026-5970`:  `golang.org/x/text` -> v0.39.0

### Known limitations
- Three advisories against `github.com/hamba/avro/v2` remain open: `GO-2026-5046`
  (CPU exhaustion), `GO-2026-5047` (integer overflow), and `GO-2026-5048` (denial of
  service via unbounded map allocations). No fixed version exists — upstream has not
  released past v2.31.0, and the `2.33.0` fix referenced by these advisories applies
  to a fork (`github.com/iskorotkov/avro/v2`), not to this dependency. The affected
  decoder is reachable from the query path, so the mitigating factor is that the Avro
  input is the response from the configured Kinetica server rather than untrusted
  data; the impact would be denial of service in the plugin backend process. Tracking
  upstream for a release.
- Remaining `npm audit` findings resolve to `@grafana/ui`, `@grafana/data`, and
  `@grafana/runtime`, which are webpack externals supplied by the Grafana host at
  runtime, plus jest/eslint build tooling. None of that code is bundled into the
  shipped `dist/module.js`. Fixing them requires major upgrades of the `@grafana/*`
  packages to 13.x, which would raise the minimum supported Grafana version beyond
  the declared `>=12.3.0`.


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
- Security issues (Go)
  - `GO-2026-6061` / `GHSA-hrxh-6v49-42gf`:  `google.golang.org/grpc` -> v1.82.1
    (xDS RBAC authorization engine and HTTP/2 transport server vulnerabilities)
- Security issues (npm, lockfile only — no declared dependency ranges changed)
  - `CVE-2026-13676`, `CVE-2026-16221`:  `fast-uri` -> 3.1.4
    (host confusion via literal backslash authority delimiter, and via failed IDN
    canonicalization)
  - `CVE-2026-59869`:  `js-yaml` -> 3.15.0 / 4.3.0
    (quadratic-complexity denial of service in YAML merge-key handling)
  - `GHSA-r28c-9q8g-f849`:  `postcss` -> 8.5.23
    (path traversal in previous-source-map auto-loading via `sourceMappingURL`)
  - `GHSA-8988-4f7v-96qf`:  `@opentelemetry/core` -> 2.8.0
    (unbounded memory allocation in W3C Baggage propagation)


## [1.0.7] - 2026-06-17 - https://github.com/kineticadb/grafana-kinetica-datasource/releases/tag/v1.0.7

Initial release of the Kinetica Grafana Datasource Plugin.

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
