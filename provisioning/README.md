# Grafana Provisioning

This directory contains provisioning configurations for automatically setting up datasources and dashboards when Grafana starts.

For more information see [Provision dashboards and data sources](https://grafana.com/tutorials/provision-dashboards-and-data-sources/)

## Directory Structure

```
provisioning/
├── README.md
├── dashboards/
│   ├── dashboards.yml          # Dashboard provider configuration
│   └── kinetica-sample-dashboard.json  # Sample dashboard
└── datasources/
    └── datasources.yml         # Kinetica datasource configuration
```

## Prerequisites

Before starting the test environment, create a `.env` file in the plugin root directory:

```bash
# .env
KINETICA_URL=http://your-kinetica-host:9191
KINETICA_USER=your_username
KINETICA_PASSWORD=your_password
```

## Quick Start

1. **Create environment file**:
   ```bash
   cp .env.example .env
   # Edit .env with your Kinetica connection details
   ```

2. **Build the plugin**:
   ```bash
   npm install
   npm run build
   ~/go/bin/mage BuildBackend
   ```

3. **Start Grafana**:
   ```bash
   docker compose up -d
   ```

4. **Access Grafana**:
   - URL: http://localhost:3001
   - Username: `admin`
   - Password: `admin`

5. **View sample dashboard**:
   - Navigate to Dashboards > Kinetica > Kinetica Sample Dashboard
   - Edit the queries to use your actual tables

## Datasource Configuration

The datasource is automatically provisioned from `datasources/datasources.yml`:

| Setting | Value | Description |
|---------|-------|-------------|
| Name | `datasource` | Default datasource name |
| Type | `kinetica-datasource` | Plugin ID |
| URL | `${KINETICA_URL}` | Kinetica REST API endpoint |
| Username | `${KINETICA_USER}` | Database username |
| Password | `${KINETICA_PASSWORD}` | Database password (secure) |

## Sample Dashboard

The sample dashboard (`kinetica-sample-dashboard.json`) includes:

| Panel | Type | Description |
|-------|------|-------------|
| Time Series Example | timeseries | Demonstrates time-based queries |
| Table Example | table | Shows raw query results |
| Aggregation Example | piechart | GROUP BY with COUNT |
| Total Records | stat | Single value aggregation |
| Bar Chart Example | barchart | Grouped bar visualization |
| Query Builder Example | table | Visual query builder usage |

### Customizing the Dashboard

1. Edit the dashboard in Grafana UI
2. Replace placeholder queries with your actual tables:
   - `your_schema.your_table` → your actual table
   - `timestamp_column` → your time column
   - `value_column` → your metric column
   - `category_column` → your dimension column

3. Export the updated dashboard:
   - Dashboard Settings > JSON Model > Copy to clipboard
   - Save to `provisioning/dashboards/`

## Adding Custom Dashboards

1. Create your dashboard in Grafana UI
2. Export as JSON:
   - Go to Dashboard Settings (gear icon)
   - Select "JSON Model"
   - Copy the JSON content

3. Save to `provisioning/dashboards/your-dashboard.json`

4. Ensure the dashboard uses the datasource variable:
   ```json
   "datasource": {
     "type": "kinetica-datasource",
     "uid": "${DS_KINETICA}"
   }
   ```

5. Restart Grafana:
   ```bash
   docker compose restart grafana
   ```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `KINETICA_URL` | Yes | Kinetica REST API URL (e.g., `http://host:9191`) |
| `KINETICA_USER` | Yes | Database username |
| `KINETICA_PASSWORD` | Yes | Database password |

## Troubleshooting

### Datasource not appearing

1. Check container logs:
   ```bash
   docker logs kinetica-grafana-dev
   ```

2. Verify environment variables are set:
   ```bash
   docker compose config
   ```

3. Restart Grafana:
   ```bash
   docker compose restart grafana
   ```

### Dashboard not loading

1. Check provisioning logs in Grafana
2. Verify JSON syntax:
   ```bash
   cat provisioning/dashboards/kinetica-sample-dashboard.json | jq .
   ```

3. Ensure dashboard provider is configured in `dashboards.yml`

### Connection errors

1. Verify Kinetica is accessible from Docker:
   ```bash
   docker exec kinetica-grafana-dev curl -s http://host.docker.internal:9191/health
   ```

2. Check network configuration in `docker-compose.yaml`

3. For local Kinetica, use `host.docker.internal` instead of `localhost`
