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
  test('should render config editor with all required fields', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    page
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    await createDataSourceConfigPage({ type: ds.type });

    // Verify Kinetica config fields are visible
    await expect(page.getByPlaceholder('http://<host>:9191')).toBeVisible();
    await expect(page.getByText('URL')).toBeVisible();
    await expect(page.getByText('User')).toBeVisible();
    await expect(page.getByText('Password')).toBeVisible();
  });

  test('should be able to fill in datasource configuration', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    page,
  }) => {
    const ds = await readProvisionedDataSource<KineticaDataSourceOptions, KineticaSecureJsonData>({
      fileName: 'datasources.yml'
    });
    const configPage = await createDataSourceConfigPage({ type: ds.type });

    // Fill in Kinetica connection settings
    await page.getByPlaceholder('http://<host>:9191').fill('http://localhost:9191');

    // User field - find by proximity to label
    const userInput = page.locator('input').filter({ hasNot: page.locator('[type="password"]') }).nth(2);
    await userInput.fill('test_user');

    // Password field
    await page.locator('input[type="password"]').fill('test_password');

    // Verify the Save & test button is visible and enabled
    const saveButton = page.getByRole('button', { name: 'Save & test' });
    await expect(saveButton).toBeVisible();
    await expect(saveButton).toBeEnabled();
  });

  test('"Save & test" should fail when configuration is invalid', async ({
    createDataSourceConfigPage,
    readProvisionedDataSource,
    page,
  }) => {
    const ds = await readProvisionedDataSource<KineticaDataSourceOptions, KineticaSecureJsonData>({
      fileName: 'datasources.yml'
    });
    const configPage = await createDataSourceConfigPage({ type: ds.type });

    // Fill in invalid URL to test error handling
    await page.getByPlaceholder('http://<host>:9191').fill('http://invalid-host:9191');

    // User field
    const userInput = page.locator('input').filter({ hasNot: page.locator('[type="password"]') }).nth(2);
    await userInput.fill('invalid_user');

    // Password field
    await page.locator('input[type="password"]').fill('invalid_password');

    // Save & test should fail with invalid configuration
    await expect(configPage.saveAndTest()).not.toBeOK();
  });

  test('provisioned datasource should be accessible', async ({
    readProvisionedDataSource,
    gotoDataSourceConfigPage,
    page,
  }) => {
    // Read the provisioned datasource
    const datasource = await readProvisionedDataSource({ fileName: 'datasources.yml' });

    // Navigate to the datasource config page
    await gotoDataSourceConfigPage(datasource.uid);

    // Verify we can access the provisioned datasource config page
    // Note: saveAndTest() may fail if Kinetica is not running, which is expected
    // The key test is that we can navigate to and load the provisioned datasource
    await expect(page.getByPlaceholder('http://<host>:9191')).toBeVisible();
  });
});
