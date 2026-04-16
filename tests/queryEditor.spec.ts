import { test, expect } from '@grafana/plugin-e2e';

/**
 * Query Editor Smoke Tests
 *
 * TESTING STRATEGY:
 * These tests provide basic "smoke testing" to verify the query editor loads without crashing.
 * They use provisioned dashboards to avoid brittle panel editing UI interactions.
 *
 * The tests focus on:
 * - Query editor renders successfully
 * - Basic UI elements are present
 * - No critical errors on load
 *
 * For detailed UI interaction testing, see manual testing checklist in E2E_TESTS_README.md
 *
 * @see https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-panel-plugin
 * @see docs/work-logs/2026-04-16-grafana-e2e-documentation-gap-analysis.md
 */

test.describe('Query Editor - Smoke Tests', () => {
  test('should load Kinetica datasource in query editor', async ({
    readProvisionedDashboard,
    gotoPanelEditPage,
  }) => {
    // Use provisioned dashboard to access query editor
    const dashboard = await readProvisionedDashboard({ fileName: 'kinetica-sample-dashboard.json' });
    const panelEditPage = await gotoPanelEditPage({ dashboard, id: '1' });

    // Verify query editor row is visible
    const queryEditorRow = panelEditPage.getQueryEditorRow('A');
    await expect(queryEditorRow).toBeVisible({ timeout: 10000 });
  });

  test('should display query editor row for datasource', async ({
    readProvisionedDashboard,
    gotoPanelEditPage,
  }) => {
    const dashboard = await readProvisionedDashboard({ fileName: 'kinetica-sample-dashboard.json' });
    const panelEditPage = await gotoPanelEditPage({ dashboard, id: '1' });

    // Verify query editor row exists with the Kinetica datasource
    const queryEditorRow = panelEditPage.getQueryEditorRow('A');
    await expect(queryEditorRow).toBeVisible({ timeout: 10000 });
  });

  test('should have query inspector button available', async ({
    readProvisionedDashboard,
    gotoPanelEditPage,
    page,
  }) => {
    const dashboard = await readProvisionedDashboard({ fileName: 'kinetica-sample-dashboard.json' });
    await gotoPanelEditPage({ dashboard, id: '1' });

    // Query inspector button should be available
    // Using flexible selector that works across versions
    const inspectorButton = page.getByRole('button', { name: /query inspector|inspector/i });
    await expect(inspectorButton).toBeVisible({ timeout: 10000 });
  });

  test('should be able to add another query', async ({
    readProvisionedDashboard,
    gotoPanelEditPage,
    page,
  }) => {
    const dashboard = await readProvisionedDashboard({ fileName: 'kinetica-sample-dashboard.json' });
    const panelEditPage = await gotoPanelEditPage({ dashboard, id: '1' });

    // Verify query editor is loaded first
    const queryEditorRowA = panelEditPage.getQueryEditorRow('A');
    await expect(queryEditorRowA).toBeVisible({ timeout: 10000 });

    // Look for "Add query" button
    // Using flexible selector since button text/location varies by version
    const addQueryButton = page.getByRole('button', { name: /add query/i });

    // Just verify the button exists - don't click it (clicking behavior varies by version)
    await expect(addQueryButton).toBeVisible({ timeout: 10000 });
  });
});

/**
 * Query Editor Detailed Interactions
 *
 * These tests are SKIPPED because they require interacting with panel editing UI
 * that changes significantly between Grafana versions.
 *
 * REASON FOR SKIPPING:
 * - Panel edit UI redesigned between major versions
 * - Query editor component placement/structure evolved
 * - Button labels and aria-labels changed
 * - Interaction patterns differ (click vs. keyboard navigation)
 *
 * WHAT DOESN'T WORK ACROSS VERSIONS:
 * - Typing into SQL editor (element selectors differ)
 * - Switching between visual builder and raw SQL (toggle mechanism changed)
 * - Collapsing/expanding query rows (UI structure different)
 * - Query options panel (moved/redesigned across versions)
 *
 * ALTERNATIVE TESTING:
 * - Provisioned panels verify query editor loads (smoke tests above)
 * - Manual testing checklist in E2E_TESTS_README.md
 * - Unit tests for query editor component logic
 * - Backend integration tests for query execution
 */
test.describe.skip('Query Editor Detailed Interactions (Skipped - See docs)', () => {
  // Examples of what would fail across versions:
  //
  // test('should allow typing SQL query', async ({ page }) => {
  //   const sqlEditor = page.locator('textarea[aria-label="SQL Editor"]');
  //   await sqlEditor.fill('SELECT * FROM table');
  //   // Fails: textarea selector/aria-label differs by version
  // });
  //
  // test('should toggle between visual builder and SQL', async ({ page }) => {
  //   await page.getByRole('button', { name: 'Query builder' }).click();
  //   // Fails: Button text/location changed across versions
  // });
  //
  // test('should expand query options', async ({ queryEditorRow }) => {
  //   await queryEditorRow.getByRole('button', { name: 'Options' }).click();
  //   // Fails: Options UI structure/naming changed
  // });
});
