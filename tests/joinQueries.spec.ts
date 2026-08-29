import { test, expect } from '@grafana/plugin-e2e';

const DASHBOARD = '/d/kinetica-var-smoke?orgId=1';
const DS_UID = 'kinetica-grafana-datasource-test';

/**
 * Covers a query builder join whose table lives in a different schema from the base
 * table. Like the variable tests these need a live Kinetica, so they self-skip when the
 * health check fails, as in CI.
 *
 * The fixture joins ki_catalog to information_schema -- both Kinetica system schemas,
 * so it runs against any instance without seeding data.
 */
test.describe('Kinetica cross-schema joins', () => {
  test.beforeEach(async ({ request }) => {
    const res = await request.get(`/api/datasources/uid/${DS_UID}/health`);
    const healthy = res.ok() && (await res.json()).status === 'OK';
    test.skip(!healthy, 'requires a live Kinetica behind the provisioned datasource');
  });

  test('renders a join against another schema', async ({ page }) => {
    // viewPanel renders it alone: the panel sits below the fold on the dashboard and
    // Grafana does not render off-screen panels.
    await page.goto(`${DASHBOARD}&viewPanel=3`, { waitUntil: 'networkidle' });
    await expect(page.getByText('Cross-schema join').first()).toBeVisible({ timeout: 20000 });
    // Every Kinetica instance has these schemas, so the join must return rows.
    await expect(page.getByText('information_schema').first()).toBeVisible({ timeout: 20000 });
  });

  test('offers the join schema and its own tables in the builder', async ({ page }) => {
    const tableFetches: string[] = [];
    page.on('request', (r) => {
      const u = r.url();
      if (u.includes('/resources/tables')) { tableFetches.push(decodeURIComponent(u.split('schema=')[1] ?? '')); }
    });

    await page.goto(`${DASHBOARD}&editPanel=3`, { waitUntil: 'networkidle' });

    // Regression: one shared table list meant a join could only ever offer the base
    // schema's tables, and the join's own schema had no control at all.
    await expect.poll(() => tableFetches, { timeout: 20000 }).toEqual(
      expect.arrayContaining(['ki_catalog', 'information_schema'])
    );
    await expect(page.getByText('Join Schema').first()).toBeVisible();
  });
});
