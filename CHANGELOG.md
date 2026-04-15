# Changelog

All notable changes to the Kinetica Grafana Datasource Plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- E2E test suite with 34 Playwright tests
- Test environment documentation
- Plugin validation documentation
- Publishing compliance report

### Changed
- Updated documentation to follow Grafana publishing best practices

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
