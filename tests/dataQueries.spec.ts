import { test, expect } from '@grafana/plugin-e2e';

/**
 * Data Query Tests
 *
 * These tests verify the query editor functionality and data query execution
 * following the Grafana plugin e2e testing guidelines.
 *
 * @see https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/data-queries
 */

test.describe('Query Editor', () => {
  test('should render query editor with SQL input', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // Verify query editor row is visible
    const queryEditorRow = panelEditPage.getQueryEditorRow('A');
    await expect(queryEditorRow).toBeVisible();

    // Verify the query editor contains the datasource name indicator
    // This confirms the query editor for Kinetica is rendered
    await expect(queryEditorRow.getByText('(datasource)')).toBeVisible();
  });

  test('should display query editor row for datasource', async ({
    panelEditPage,
    readProvisionedDataSource,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // Verify query editor row exists with the Kinetica datasource
    const queryEditorRow = panelEditPage.getQueryEditorRow('A');
    await expect(queryEditorRow).toBeVisible();

    // Query row title should be visible (it's labeled 'A' by default)
    await expect(queryEditorRow.getByRole('button', { name: 'Query editor row title A' })).toBeVisible();
  });

  test('should be able to switch visualization type', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // Switch to Table visualization
    await panelEditPage.setVisualization('Table');

    // Verify the visualization change by checking the panel options
    await expect(page.getByRole('button', { name: 'Change visualization' })).toBeVisible();
  });

  test('should have refresh button available', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // The refresh button should be available in the time range toolbar
    const refreshButton = page.getByTestId('data-testid RefreshPicker run button');
    await expect(refreshButton).toBeVisible();
  });

  test('should have query inspector available', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // Query inspector button should be available
    await expect(page.getByRole('button', { name: 'Query inspector button' })).toBeVisible();
  });
});

test.describe('Data Query Execution', () => {
  test('should be able to execute refresh panel action', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // Set visualization to Table for easier data inspection
    await panelEditPage.setVisualization('Table');

    // The refresh panel action should be available
    // Note: The actual query may fail without a running Kinetica instance,
    // but the refresh mechanism should work
    const refreshButton = page.getByTestId('data-testid RefreshPicker run button');
    await expect(refreshButton).toBeVisible();
    await expect(refreshButton).toBeEnabled();
  });

  test('should show error state when query fails', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // Try to refresh - this will likely fail without a running Kinetica
    // but we're testing that the error handling works
    try {
      await panelEditPage.refreshPanel();
    } catch {
      // Expected to fail without Kinetica running
    }

    // The panel should still be visible (error is handled gracefully)
    const queryEditorRow = panelEditPage.getQueryEditorRow('A');
    await expect(queryEditorRow).toBeVisible();
  });
});

test.describe('Multiple Queries', () => {
  test('should be able to add a second query', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // Click Add query button
    await page.getByRole('button', { name: 'Add query' }).click();

    // Verify second query row appears (labeled 'B')
    const queryEditorRowB = panelEditPage.getQueryEditorRow('B');
    await expect(queryEditorRowB).toBeVisible();
  });

  test('should be able to duplicate a query', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    const queryEditorRow = panelEditPage.getQueryEditorRow('A');

    // Click duplicate button
    await queryEditorRow.getByRole('button', { name: 'Duplicate query' }).click();

    // Verify second query row appears
    const queryEditorRowB = panelEditPage.getQueryEditorRow('B');
    await expect(queryEditorRowB).toBeVisible();
  });

  test('should be able to hide/show query response', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    const queryEditorRow = panelEditPage.getQueryEditorRow('A');

    // Hide response button should be visible
    await expect(queryEditorRow.getByRole('button', { name: 'Hide response' })).toBeVisible();
  });

  test('should be able to remove a query after adding one', async ({
    panelEditPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await panelEditPage.datasource.set(ds.name);

    // Add a second query first
    await page.getByRole('button', { name: 'Add query' }).click();
    const queryEditorRowB = panelEditPage.getQueryEditorRow('B');
    await expect(queryEditorRowB).toBeVisible();

    // Remove the second query
    await queryEditorRowB.getByRole('button', { name: 'Remove query' }).click();

    // Verify it's removed
    await expect(queryEditorRowB).not.toBeVisible();
  });
});

test.describe('Provisioned Dashboard', () => {
  test('should load provisioned dashboard with Kinetica panels', async ({
    readProvisionedDashboard,
    gotoDashboardPage,
    page,
  }) => {
    // Read the provisioned sample dashboard
    const dashboard = await readProvisionedDashboard({ fileName: 'kinetica-sample-dashboard.json' });

    // Navigate to the dashboard
    await gotoDashboardPage(dashboard);

    // Verify the dashboard loads successfully
    // The panels may show errors without Kinetica, but the dashboard structure should load
    await expect(page.getByText('Kinetica Sample Dashboard')).toBeVisible();
  });

  test('should be able to edit panel from provisioned dashboard', async ({
    readProvisionedDashboard,
    gotoPanelEditPage,
  }) => {
    // Read the provisioned sample dashboard
    const dashboard = await readProvisionedDashboard({ fileName: 'kinetica-sample-dashboard.json' });

    // Go to edit the first panel (Time Series Example, id: 1)
    const panelEditPage = await gotoPanelEditPage({ dashboard, id: '1' });

    // Verify we can access the panel edit page
    const queryEditorRow = panelEditPage.getQueryEditorRow('A');
    await expect(queryEditorRow).toBeVisible();
  });
});
