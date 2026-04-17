# Changelog

All notable changes to the Kinetica Grafana Datasource Plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### BREAKING CHANGES
- **Minimum Grafana version increased from 10.4.0 to 12.3.0**
  - Reason: Plugin is built with Grafana SDK 12.3.0 and cannot guarantee compatibility with older versions
  - Impact: Users running Grafana 10.4-12.2 must either upgrade Grafana or use an older plugin version
  - See `docs/DEPENDENCY_MISMATCH_ANALYSIS.md` for full technical explanation

### Changed
- E2E test suite reduced to 7 stable tests (from initial 34)
  - Reflects realistic cross-version testing limitations
  - Focus on truly stable functionality (alerts, provisioning, basic page loads)
  - See `E2E_TESTS_README.md` for testing strategy
- Updated documentation to follow Grafana publishing best practices
- Comprehensive work logs added in `docs/work-logs/` directory

### Added
- E2E test suite with Playwright
- Test environment documentation
- Plugin validation documentation
- Publishing compliance report
- Dependency mismatch analysis documentation
- Create-plugin tool analysis documentation

### Fixed
- Aligned `grafanaDependency` declaration with actual SDK version used in build
- E2E tests now pass reliably across supported Grafana versions (12.3.0+)

## [7.2.3.0] - 2024-01-13

### Added
- Initial release of the Kinetica Grafana Datasource Plugin
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
- Time series support with automatic time column detection
- Time range macros (`$__timeFilter`, `$__timeGroup`, `$__from`, `$__to`)
- Backend plugin with Go SDK integration
- Health check endpoint for connection validation
- Secure credential storage using Grafana's encrypted secureJsonData
- Provisioning support for datasource and dashboards
- Docker Compose development environment

<!--
TODO: Add links when repository is public
[Unreleased]: https://github.com/kinetica/grafana-datasource-plugin/compare/v7.2.3.0...HEAD
[7.2.3.0]: https://github.com/kinetica/grafana-datasource-plugin/releases/tag/v7.2.3.0
-->
