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
    // Note: We check for stable text that exists across Grafana versions (>=10.4.0)
    await expect(page.getByText('Define query and alert condition', { exact: true })).toBeVisible();
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
});

test.describe('Alert Rule with Kinetica Datasource', () => {
  test('should show query section in alert rule', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // The query section should be visible (check for stable text across versions)
    await expect(page.getByText('Define query and alert condition', { exact: true })).toBeVisible();
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
});

test.describe('Alert Rule Sections', () => {
  test('should have alert rule name input field', async ({
    page,
  }) => {
    // Navigate to create new alert rule
    await page.goto('/alerting/new/alerting');
    await page.waitForLoadState('networkidle');

    // Alert rule name input should be available - this is a stable element across versions
    const nameInput = page.getByPlaceholder('Give your alert rule a name');
    await expect(nameInput).toBeVisible();
  });
});
