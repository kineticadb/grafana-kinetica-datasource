# Kinetica Datasource Plugin - E2E Tests

This directory contains end-to-end tests for the Kinetica Grafana datasource plugin, following the [Grafana Plugin E2E Testing Guide](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/).

## Test Framework

- **Playwright** - Browser automation and testing
- **@grafana/plugin-e2e** - Grafana-specific testing utilities and fixtures

## Test Structure

```
tests/
├── README.md                 # This documentation
├── alertQueries.spec.ts      # Alert Query tests (12 tests)
├── configEditor.spec.ts      # Config Editor tests (4 tests)
├── dataQueries.spec.ts       # Data Query tests (14 tests)
└── queryEditor.spec.ts       # Query Editor tests (4 tests)
```

**Total: 34 tests**

## Test Coverage

The test suite covers all applicable test types from the [Grafana Plugin E2E Testing Guide](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/):

| Test Type | Status | Tests | Reference |
|-----------|--------|-------|-----------|
| Configuration Tests | ✅ Complete | 4 | [configurations](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/configurations) |
| Data Query Tests | ✅ Complete | 14 | [data-queries](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/data-queries) |
| Query Editor Tests | ✅ Complete | 4 | Part of data queries |
| Alert Query Tests | ✅ Complete | 12 | [alert-queries](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/alert-queries) |
| Variable Query Tests | ⚠️ N/A | - | Plugin lacks custom variable editor |
| Annotation Query Tests | ⚠️ N/A | - | Plugin lacks annotation support |

### Config Editor Tests (`configEditor.spec.ts`)

Based on [Grafana configuration test guidelines](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/configurations):

| Test | Grafana Doc Requirement | Description |
|------|------------------------|-------------|
| `should render config editor with all required fields` | Config editor load | Verifies URL, User, Password fields are visible |
| `should be able to fill in datasource configuration` | Valid configuration | Fills connection settings, verifies Save & test enabled |
| `"Save & test" should fail when configuration is invalid` | Invalid configuration error | Tests error handling with invalid connection |
| `provisioned datasource should be accessible` | Provisioned datasource | Verifies provisioned datasource config page loads |

### Query Editor Tests (`queryEditor.spec.ts`)

| Test | Description |
|------|-------------|
| `smoke: should load Kinetica datasource in query editor` | Datasource loads in panel edit mode |
| `should display query editor row for datasource` | Query editor row renders with proper title |
| `should have query inspector button available` | Query inspector button is present |
| `should be able to add another query` | Add query button functionality |

### Data Query Tests (`dataQueries.spec.ts`)

Based on [Grafana data query test guidelines](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/data-queries):

**Query Editor Tests:**

| Test | Grafana Doc Requirement | Description |
|------|------------------------|-------------|
| `should render query editor with SQL input` | Query editor render | Verifies query editor renders with input field |
| `should display query editor row for datasource` | Query row visible | Query editor row renders with title |
| `should be able to switch visualization type` | Visualization switch | Tests switching to Table visualization |
| `should have refresh button available` | Refresh panel | Refresh button is visible |
| `should have query inspector available` | Query inspector | Query inspector button present |

**Data Query Execution Tests:**

| Test | Grafana Doc Requirement | Description |
|------|------------------------|-------------|
| `should be able to execute refresh panel action` | Query execution | Refresh button is enabled |
| `should show error state when query fails` | Error handling | Panel remains visible on query error |

**Multiple Queries Tests:**

| Test | Description |
|------|-------------|
| `should be able to add a second query` | Add second query (B) |
| `should be able to duplicate a query` | Query duplication |
| `should be able to hide/show query response` | Hide response button |
| `should be able to remove a query after adding one` | Query removal |

**Provisioned Dashboard Tests:**

| Test | Grafana Doc Requirement | Description |
|------|------------------------|-------------|
| `should load provisioned dashboard with Kinetica panels` | Dashboard query validation | Provisioned dashboard loads |
| `should be able to edit panel from provisioned dashboard` | Panel edit from dashboard | Panel edit page accessible |

### Alert Query Tests (`alertQueries.spec.ts`)

Based on [Grafana alert query test guidelines](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/alert-queries):

**Alert Rule Page Tests:**

| Test | Description |
|------|-------------|
| `should load alert rule creation page successfully` | Alert rule page loads |
| `should have evaluation behavior section` | Evaluation behavior section visible |
| `should have pending period configuration` | Pending period option available |
| `should have configure notifications section` | Notifications section visible |
| `should have save and cancel buttons` | Save/Cancel buttons present |

**Alert Rule with Kinetica Datasource Tests:**

| Test | Description |
|------|-------------|
| `should show query section in alert rule` | Query section visible |
| `should have alert condition configuration` | Alert condition config visible |
| `should have preview alert rule condition button` | Preview button available |

**Alert Rule Sections Tests:**

| Test | Description |
|------|-------------|
| `should show enter alert rule name section` | Name section visible |
| `should show folder and labels section` | Folder/labels section visible |
| `should show notification message configuration` | Notification message section |
| `should have alert rule name input field` | Name input field present |

## Prerequisites

1. **Node.js** >= 22
2. **Docker** and Docker Compose
3. **Playwright browsers** installed

## Running Tests

### 1. Install Dependencies

```bash
npm install
```

### 2. Install Playwright Browsers

```bash
npx playwright install chromium
```

### 3. Build the Plugin

```bash
npm run build
```

### 4. Start Grafana Container

```bash
docker compose up -d
```

Wait for Grafana to be healthy:

```bash
curl http://localhost:3001/api/health
```

### 5. Run E2E Tests

```bash
npm run e2e
```

## Test Configuration

### Playwright Config (`playwright.config.ts`)

Key settings:
- **Base URL**: `http://localhost:3001` (Grafana instance)
- **Browser**: Chromium
- **Authentication**: Stored in `playwright/.auth/admin.json`
- **Reporter**: HTML report

### Docker Compose

The test environment uses Docker Compose to run Grafana with the plugin:

- **Port**: 3001 (mapped to container's 3000)
- **Plugin**: Mounted from `./dist` directory
- **Provisioning**: Auto-loads datasource and dashboards from `./provisioning/`

### Provisioned Resources

**Datasource** (`provisioning/datasources/datasources.yml`):

```yaml
datasources:
  - name: datasource
    uid: kinetica-datasource-test
    type: kinetica-datasource
    url: ${KINETICA_URL}
    jsonData:
      username: ${KINETICA_USER}
    secureJsonData:
      password: ${KINETICA_PASSWORD}
```

**Dashboard** (`provisioning/dashboards/kinetica-sample-dashboard.json`):
- Sample dashboard with 7 panels demonstrating plugin capabilities
- Used for provisioned dashboard tests

## Writing New Tests

### Config Editor Test Pattern

```typescript
test('config editor test', async ({
  createDataSourceConfigPage,
  readProvisionedDataSource,
  page
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  const configPage = await createDataSourceConfigPage({ type: ds.type });

  // Fill configuration
  await page.getByPlaceholder('http://<host>:9191').fill('http://localhost:9191');

  // Verify save
  await expect(configPage.saveAndTest()).toBeOK();
});
```

### Query Editor Test Pattern

```typescript
test('query editor test', async ({
  panelEditPage,
  readProvisionedDataSource
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await panelEditPage.datasource.set(ds.name);

  const queryEditorRow = panelEditPage.getQueryEditorRow('A');
  await expect(queryEditorRow).toBeVisible();
});
```

### Data Query Test Pattern

```typescript
test('data query test', async ({
  panelEditPage,
  readProvisionedDataSource,
  page
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await panelEditPage.datasource.set(ds.name);

  // Set visualization
  await panelEditPage.setVisualization('Table');

  // Execute query (may fail without running Kinetica)
  await expect(panelEditPage.refreshPanel()).toBeOK();
});
```

### Alert Query Test Pattern

```typescript
test('alert query test', async ({ page }) => {
  await page.goto('/alerting/new/alerting');
  await page.waitForLoadState('networkidle');

  // Verify alert sections
  await expect(page.getByText('Define query and alert condition', { exact: true })).toBeVisible();
});
```

### Provisioned Dashboard Test Pattern

```typescript
test('provisioned dashboard test', async ({
  readProvisionedDashboard,
  gotoDashboardPage,
  page
}) => {
  const dashboard = await readProvisionedDashboard({ fileName: 'kinetica-sample-dashboard.json' });
  await gotoDashboardPage(dashboard);

  await expect(page.getByText('Kinetica Sample Dashboard')).toBeVisible();
});
```

## Debugging Tests

### Run Tests with UI

```bash
npx playwright test --ui
```

### Run Single Test

```bash
npm run e2e -- --grep "should render config editor"
```

### View Test Report

```bash
npx playwright show-report
```

### Debug Mode

```bash
npx playwright test --debug
```

## Troubleshooting

### Port Conflicts

If port 3001 is in use, modify the port in:
- `docker-compose.yaml`
- `.config/docker-compose-base.yaml`
- `playwright.config.ts`

### Authentication Issues

Delete stored auth state and re-run:

```bash
rm -rf playwright/.auth
npm run e2e
```

### Plugin Not Loading

1. Ensure plugin is built: `npm run build`
2. Restart Grafana: `docker compose restart grafana`
3. Check logs: `docker logs kinetica-grafana-dev`

### Provisioning Errors

If you see "data source not found" errors:
1. Ensure `.env` file exists with valid values
2. Restart fresh: `docker compose down && docker compose up -d`

## References

- [Grafana Plugin E2E Testing Guide](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/)
- [Configuration Tests](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/configurations)
- [Data Query Tests](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/data-queries)
- [Alert Query Tests](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/alert-queries)
- [@grafana/plugin-e2e Package](https://www.npmjs.com/package/@grafana/plugin-e2e)
- [Playwright Documentation](https://playwright.dev/docs/intro)
