# Test Environment Setup

This document describes how to set up a test environment for the Kinetica Grafana datasource plugin, following the [Grafana Plugin Test Environment Guidelines](https://grafana.com/developers/plugin-tools/publish-a-plugin/provide-test-environment).

## Overview

A properly configured test environment accelerates plugin review and helps users understand plugin functionality. The test environment includes:

- Docker Compose configuration for Grafana
- Provisioned datasource configuration
- Sample dashboard demonstrating plugin capabilities

## Quick Start

### 1. Prerequisites

- Docker and Docker Compose
- Node.js >= 22
- Go >= 1.22 (for backend)
- Access to a Kinetica database

### 2. Clone and Build

```bash
# Clone the repository
git clone https://github.com/kinetica/grafana-datasource-plugin.git
cd grafana-datasource-plugin

# Install dependencies
npm install

# Build frontend
npm run build

# Build backend (optional - binaries may be pre-built)
go install github.com/magefile/mage@latest
~/go/bin/mage BuildBackend
```

### 3. Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit with your Kinetica connection details
nano .env
```

Required environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `KINETICA_URL` | Kinetica REST API endpoint | `http://host.docker.internal:9191` |
| `KINETICA_USER` | Database username | `admin` |
| `KINETICA_PASSWORD` | Database password | `your_password` |

### 4. Start Test Environment

```bash
# Start Grafana with the plugin
docker compose up -d

# Wait for Grafana to be ready
curl http://localhost:3001/api/health
```

### 5. Access Grafana

- **URL**: http://localhost:3001
- **Username**: `admin`
- **Password**: `admin`

### 6. View Sample Dashboard

1. Navigate to **Dashboards** in the left sidebar
2. Open the **Kinetica** folder
3. Click **Kinetica Sample Dashboard**
4. Edit panels to use your actual tables

## Test Environment Components

### Docker Compose (`docker-compose.yaml`)

The Docker Compose configuration:

- Runs Grafana on port **3001** (to avoid conflicts)
- Mounts the plugin from the local `dist/` directory
- Loads provisioning from `./provisioning/`
- Allows unsigned plugins for development
- Maps `host.docker.internal` for local Kinetica access

```yaml
services:
  grafana:
    container_name: kinetica-grafana-dev
    ports:
      - "3001:3000"
    environment:
      - GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=kinetica-datasource
    extra_hosts:
      - "host.docker.internal:host-gateway"
    volumes:
      - ./dist:/var/lib/grafana/plugins/kinetica-datasource
      - ./provisioning:/etc/grafana/provisioning
```

### Datasource Provisioning (`provisioning/datasources/datasources.yml`)

Automatically configures the Kinetica datasource:

```yaml
apiVersion: 1

datasources:
  - name: datasource
    type: kinetica-datasource
    access: proxy
    url: ${KINETICA_URL}
    isDefault: true
    jsonData:
      username: ${KINETICA_USER}
      tlsSkipVerify: true
    secureJsonData:
      password: ${KINETICA_PASSWORD}
```

### Dashboard Provisioning (`provisioning/dashboards/`)

Provides a sample dashboard with:

| Panel | Visualization | SQL Pattern |
|-------|---------------|-------------|
| Time Series | Line chart | `SELECT time, value FROM table WHERE $__timeFilter(time)` |
| Table View | Table | `SELECT * FROM table LIMIT 100` |
| Aggregation | Pie chart | `SELECT category, COUNT(*) FROM table GROUP BY category` |
| Statistics | Stat | `SELECT COUNT(*) FROM table` |
| Bar Chart | Bar | `SELECT category, SUM(value) FROM table GROUP BY category` |
| Query Builder | Table | Visual query builder example |

## Customizing for Your Environment

### Using Your Own Tables

1. Edit the sample dashboard in Grafana
2. Replace placeholder queries:
   ```sql
   -- Before
   SELECT * FROM your_schema.your_table

   -- After
   SELECT * FROM ki_home.sensor_data
   ```

3. Save the dashboard
4. Export to `provisioning/dashboards/` for persistence

### Adding Demo Data

If you have a Kinetica instance with demo data:

1. Update the sample dashboard queries to use demo tables
2. Export the working dashboard as JSON
3. Replace `kinetica-sample-dashboard.json`

Example with demo data:

```sql
-- Time series from sensor data
SELECT
    timestamp AS time,
    temperature AS value
FROM demo.sensor_readings
WHERE $__timeFilter(timestamp)
ORDER BY timestamp

-- Aggregation by location
SELECT
    location,
    AVG(temperature) AS avg_temp
FROM demo.sensor_readings
GROUP BY location
```

### Creating Additional Dashboards

1. Build dashboards in Grafana UI
2. Export as JSON (Dashboard Settings > JSON Model)
3. Save to `provisioning/dashboards/`
4. Use datasource variable for portability:
   ```json
   "datasource": {
     "type": "kinetica-datasource",
     "uid": "${DS_KINETICA}"
   }
   ```

## Verification Checklist

Before submitting the plugin for review, verify:

- [ ] `docker compose up` starts Grafana successfully
- [ ] Datasource is auto-configured and appears in Connections
- [ ] Sample dashboard loads without errors
- [ ] At least one panel shows data (with valid Kinetica connection)
- [ ] Query editor renders correctly
- [ ] Time range picker works with time-filtered queries

## Troubleshooting

### Grafana won't start

```bash
# Check container status
docker compose ps

# View logs
docker logs kinetica-grafana-dev

# Rebuild and restart
docker compose down
npm run build
docker compose up -d
```

### Datasource connection fails

1. Verify Kinetica is accessible:
   ```bash
   curl http://your-kinetica-host:9191/health
   ```

2. Check environment variables:
   ```bash
   docker compose config | grep KINETICA
   ```

3. For local Kinetica, ensure `host.docker.internal` resolves:
   ```bash
   docker exec kinetica-grafana-dev ping host.docker.internal
   ```

### Dashboard not provisioned

1. Check dashboard provider config:
   ```bash
   cat provisioning/dashboards/dashboards.yml
   ```

2. Verify JSON is valid:
   ```bash
   cat provisioning/dashboards/kinetica-sample-dashboard.json | jq .
   ```

3. Restart Grafana:
   ```bash
   docker compose restart grafana
   ```

### Plugin not loading

1. Check plugin is built:
   ```bash
   ls -la dist/
   ```

2. Verify plugin is mounted:
   ```bash
   docker exec kinetica-grafana-dev ls /var/lib/grafana/plugins/
   ```

3. Check Grafana logs for plugin errors:
   ```bash
   docker logs kinetica-grafana-dev 2>&1 | grep -i kinetica
   ```

## CI/CD Integration

For automated testing, add to your CI workflow:

```yaml
test-environment:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4

    - name: Build plugin
      run: |
        npm ci
        npm run build

    - name: Start test environment
      run: |
        echo "KINETICA_URL=http://mock-server:9191" > .env
        echo "KINETICA_USER=test" >> .env
        echo "KINETICA_PASSWORD=test" >> .env
        docker compose up -d

    - name: Wait for Grafana
      run: |
        timeout 60 bash -c 'until curl -s http://localhost:3001/api/health; do sleep 2; done'

    - name: Verify datasource provisioned
      run: |
        curl -u admin:admin http://localhost:3001/api/datasources | jq '.[0].name'

    - name: Run e2e tests
      run: npm run e2e
```

## References

- [Grafana Provisioning Documentation](https://grafana.com/docs/grafana/latest/administration/provisioning/)
- [Provide Test Environment Guide](https://grafana.com/developers/plugin-tools/publish-a-plugin/provide-test-environment)
- [Docker Compose for Grafana](https://grafana.com/docs/grafana/latest/setup-grafana/installation/docker/)
