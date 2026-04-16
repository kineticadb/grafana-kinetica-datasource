import { test, expect } from '@grafana/plugin-e2e';
import { KineticaDataSourceOptions, KineticaSecureJsonData } from '../src/types';

/**
 * Configuration Editor Tests
 *
 * These tests verify the datasource configuration editor functionality
 * following the Grafana plugin e2e testing guidelines.
 *
 * @see https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/configurations
 */

test.describe('Config Editor', () => {
  test('should load config editor for new datasource', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    page,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await createDataSourceConfigPage({ type: ds.type });

    // Verify config page loads by checking for stable UI elements present in all versions
    // Using role-based selectors (best practice from Grafana repo)
    await expect(page.getByRole('button', { name: 'Save & test' })).toBeVisible();
  });

  test('should access provisioned datasource config', async ({
    readProvisionedDataSource,
    gotoDataSourceConfigPage,
    page,
  }) => {
    const datasource = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await gotoDataSourceConfigPage(datasource.uid);

    // Verify the config page loads successfully
    await expect(page.getByRole('button', { name: 'Save & test' })).toBeVisible();
  });
});
