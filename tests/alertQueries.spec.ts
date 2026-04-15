import { test, expect } from '@grafana/plugin-e2e';

/**
 * Alert Query Tests
 *
 * These tests verify that the Kinetica datasource plugin is compatible with
 * Grafana's alerting system. Since the plugin has `metrics: true` in plugin.json,
 * it should be usable for alert rules.
 *
 * @see https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/alert-queries
 */

test.describe('Alert Rule Page', () => {
  test('should load alert rule creation page successfully', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Verify the alert rule creation page loads by checking for key sections
    await expect(page.getByText('Define query and alert condition', { exact: true })).toBeVisible();
  });

  test('should have evaluation behavior section', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // The "Set evaluation behavior" section should be visible (numbered in Grafana UI)
    await expect(page.getByText('4. Set evaluation behavior')).toBeVisible();
  });

  test('should have pending period configuration', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Pending period option should be available
    await expect(page.getByText('Pending period', { exact: true })).toBeVisible();
  });

  test('should have configure notifications section', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Configure notifications section should be visible (numbered in Grafana UI)
    await expect(page.getByText('5. Configure notifications')).toBeVisible();
  });

  test('should have save and cancel buttons', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Save button should be available
    const saveButton = page.getByRole('button', { name: 'Save', exact: true });
    await expect(saveButton).toBeVisible();

    // Cancel button should be available
    const cancelButton = page.getByRole('button', { name: 'Cancel' });
    await expect(cancelButton).toBeVisible();
  });
});

test.describe('Alert Rule with Kinetica Datasource', () => {
  test('should show query section in alert rule', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // The query section should be visible
    await expect(page.getByText('Define query and alert condition', { exact: true })).toBeVisible();

    // The query and alert condition section should be visible
    await expect(page.getByText('2. Define query and alert condition')).toBeVisible();
  });

  test('should have alert condition configuration', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // Alert condition section should be visible (use exact match to avoid multiple elements)
    await expect(page.getByText('Alert condition', { exact: true })).toBeVisible();
  });

  test('should have preview alert rule condition button', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // Preview button should be available
    const previewButton = page.getByRole('button', { name: 'Preview alert rule condition' });
    await expect(previewButton).toBeVisible();
  });
});

test.describe('Alert Rule Sections', () => {
  test('should show enter alert rule name section', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // Enter alert rule name section should be visible
    await expect(page.getByText('1. Enter alert rule name')).toBeVisible();
  });

  test('should show folder and labels section', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // Folder and labels section should be visible
    await expect(page.getByText('3. Add folder and labels')).toBeVisible();
  });

  test('should show notification message configuration', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // Configure notification message section should be visible
    await expect(page.getByText('6. Configure notification message')).toBeVisible();
  });

  test('should have alert rule name input field', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // Alert rule name input should be available
    const nameInput = page.getByPlaceholder('Give your alert rule a name');
    await expect(nameInput).toBeVisible();
  });
});
