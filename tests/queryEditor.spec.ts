import { test, expect } from '@grafana/plugin-e2e';

test('smoke: should load Kinetica datasource in query editor', async ({ panelEditPage, readProvisionedDataSource, page }) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await panelEditPage.datasource.set(ds.name);

  // Verify the datasource is loaded by checking the query editor row is visible
  const queryEditorRow = panelEditPage.getQueryEditorRow('A');
  await expect(queryEditorRow).toBeVisible();
  await expect(queryEditorRow.getByRole('button', { name: /Query editor row title/i })).toBeVisible();
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

test('should have query inspector button available', async ({
  panelEditPage,
  readProvisionedDataSource,
  page,
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await panelEditPage.datasource.set(ds.name);

  // Verify the Query inspector button is present (always visible in the query editor header)
  await expect(page.getByRole('button', { name: 'Query inspector button' })).toBeVisible();
});

test('should be able to add another query', async ({
  panelEditPage,
  readProvisionedDataSource,
  page,
}) => {
  const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
  await panelEditPage.datasource.set(ds.name);

  // Verify the "Add query" button is present - using page instead of panelEditPage
  await expect(page.getByRole('button', { name: 'Add query' })).toBeVisible();
});
