import { test, expect } from '@grafana/plugin-e2e';

const DASHBOARD = '/d/kinetica-var-smoke?orgId=1';
const DS_UID = 'kinetica-grafana-datasource-test';

/**
 * Exercises dashboard variables of type *Query* end to end.
 *
 * Unlike the rest of the suite these need a live Kinetica behind the provisioned
 * datasource, so they self-skip when the health check fails (as in CI, which runs
 * Grafana without a database). The variable path runs entirely in the browser --
 * metricFindQuery, KineticaVariableSupport and frameToMetricFindValues -- so an
 * API-level test cannot cover it.
 */
test.describe('Kinetica query variables', () => {
  test.beforeEach(async ({ request }) => {
    const res = await request.get(`/api/datasources/uid/${DS_UID}/health`);
    const healthy = res.ok() && (await res.json()).status === 'OK';
    test.skip(!healthy, 'requires a live Kinetica behind the provisioned datasource');
  });

  test('populates a variable from a single-column query', async ({ page }) => {
    await page.goto(DASHBOARD, { waitUntil: 'networkidle' });
    await expect
      .poll(() => new URL(page.url()).searchParams.get('var-schema'), { timeout: 20000 })
      .toBeTruthy();

    // Every Kinetica instance has these system schemas.
    const picker = page.locator('[data-testid*="Variable Value DropDown"]').first();
    await picker.click();
    const options = await page.locator('[role="option"]').allInnerTexts();
    expect(options).toEqual(expect.arrayContaining(['ki_catalog', 'information_schema']));
  });

  test('uses __text as the label and __value as the stored value', async ({ page }) => {
    await page.goto(DASHBOARD, { waitUntil: 'networkidle' });
    await expect
      .poll(() => new URL(page.url()).searchParams.get('var-col'), { timeout: 20000 })
      .toBeTruthy();

    // __text is the column name, __value its numeric position: the label shown and
    // the value stored must come from different columns.
    const submenu = (await page.locator('[data-testid*="template variables"]').allInnerTexts()).join(' ');
    expect(submenu).toContain('oid');
    expect(new URL(page.url()).searchParams.get('var-col')).toMatch(/^\d+$/);
  });

  test('resolves variables concurrently without cancelling each other', async ({ page }) => {
    // Regression: a constant requestId let each variable query abort the previous
    // one, so only the last variable on a dashboard ever resolved.
    await page.goto(DASHBOARD, { waitUntil: 'networkidle' });
    await expect
      .poll(() => {
        const p = new URL(page.url()).searchParams;
        return Boolean(p.get('var-schema')) && Boolean(p.get('var-col'));
      }, { timeout: 20000 })
      .toBe(true);

    // The panel title interpolates $schema, proving the value reached the query path.
    await expect(page.getByRole('heading', { name: /^Tables in \S+/ })).toBeVisible();
  });
});
