# Kinetica Datasource for Grafana

[![Grafana](https://img.shields.io/badge/Grafana-%3E%3D10.4.0-orange)](https://grafana.com)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

<!-- TODO: Add these dynamic badges after publishing to Grafana catalog
[![Downloads](https://img.shields.io/badge/dynamic/json?logo=grafana&color=F47A20&label=downloads&query=downloads&url=https://grafana.com/api/plugins/kinetica-datasource)](https://grafana.com/grafana/plugins/kinetica-datasource)
[![Version](https://img.shields.io/badge/dynamic/json?logo=grafana&color=F47A20&label=version&query=version&url=https://grafana.com/api/plugins/kinetica-datasource)](https://grafana.com/grafana/plugins/kinetica-datasource)
-->

Connect Grafana to [Kinetica](https://www.kinetica.com) for real-time analytics and visualization of large-scale data. This plugin enables you to query Kinetica databases using a visual query builder or raw SQL, and display results in Grafana panels.

<!-- TODO: Add screenshot
![Kinetica Query Builder](src/img/screenshots/query-builder.png)
-->

## Features

- **Visual Query Builder** - Build complex SQL queries without writing code
  - Schema and table selection with autocomplete
  - Column selection with aggregation functions (AVG, COUNT, MAX, MIN, SUM, etc.)
  - JOIN support (INNER, LEFT, RIGHT, FULL) with multiple conditions
  - WHERE and HAVING clause filters
  - GROUP BY, ORDER BY, and LIMIT
  - Set operations (UNION, INTERSECT, EXCEPT)
- **Raw SQL Mode** - Write custom SQL queries with syntax highlighting
- **Time Series Support** - Native support for time-based data with automatic time column detection
- **Table Visualizations** - Display query results in Grafana tables
- **Grafana Alerting** - Create alerts based on Kinetica query results
- **Secure Credentials** - Passwords stored securely using Grafana's encrypted storage

## Requirements

- Grafana >= 10.4.0
- Kinetica >= 7.x

## Installation

### From Grafana Catalog (Recommended)

<!-- TODO: Update after publishing -->
1. In Grafana, go to **Configuration > Plugins**
2. Search for "Kinetica"
3. Click **Install**

### Manual Installation

1. Download the latest release from the [releases page](https://github.com/kinetica/grafana-datasource-plugin/releases)
2. Extract to your Grafana plugins directory (usually `/var/lib/grafana/plugins`)
3. Restart Grafana

## Configuration

1. Go to **Configuration > Data Sources**
2. Click **Add data source**
3. Search for "Kinetica" and select it
4. Configure the connection:
   - **URL**: Your Kinetica server URL (e.g., `http://localhost:9191`)
   - **User**: Database username
   - **Password**: Database password
5. Click **Save & test** to verify the connection

<!-- TODO: Add screenshot
![Configuration](src/img/screenshots/config.png)
-->

## Usage

### Visual Query Builder

1. Create a new panel and select the Kinetica datasource
2. Use the query builder to:
   - Select a schema and table
   - Add columns with optional aggregations
   - Add JOINs to combine tables
   - Add WHERE filters
   - Configure GROUP BY and ORDER BY
3. Click **Run Query** to execute

### Raw SQL Mode

1. Toggle **Raw SQL Mode** in the query editor
2. Write your SQL query in the code editor
3. The query executes on blur or when you run the panel

### Time Series Queries

For time series visualizations:
1. Select a time column in the **Time Column** dropdown
2. Use Grafana's time range picker to filter data
3. The plugin supports `$__timeFilter()` and `$__timeGroup()` macros

### Macros

| Macro | Description |
|-------|-------------|
| `$__timeFilter(column)` | Adds time range filter based on Grafana's time picker |
| `$__timeGroup(column, interval)` | Groups data by time intervals |
| `$__from` | Start of selected time range (Unix timestamp) |
| `$__to` | End of selected time range (Unix timestamp) |

## Documentation

<!-- TODO: Update with actual documentation URL -->
For detailed documentation, visit the [Kinetica Documentation](https://docs.kinetica.com).

## Development

### Prerequisites

- Node.js >= 22
- Go >= 1.22
- Docker (for testing)
- Mage (Go build tool)

### Building

```bash
# Install frontend dependencies
npm install

# Build frontend
npm run build

# Build backend (all platforms)
mage buildAll

# Or build for current platform only
mage build
```

### Running Locally

```bash
# Start development server with hot reload
npm run dev

# In another terminal, start Grafana with the plugin
docker compose up -d
```

Access Grafana at http://localhost:3001 (default credentials: admin/admin)

### Testing

```bash
# Run frontend unit tests
npm run test:ci

# Run backend tests
mage coverage

# Run E2E tests (requires Grafana running)
npm run e2e
```

### Linting

```bash
# Lint frontend
npm run lint

# Auto-fix lint issues
npm run lint:fix

# Type check
npm run typecheck
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

<!-- TODO: Update with actual support channels -->
- [Issue Tracker](https://github.com/kinetica/grafana-datasource-plugin/issues)
- [Kinetica Community](https://community.kinetica.com)
