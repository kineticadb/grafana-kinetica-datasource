# E2E Tests

Playwright end-to-end tests for the Kinetica Grafana datasource plugin, built on
[@grafana/plugin-e2e](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/test-a-data-source-plugin/).

See [`E2E_TESTS_README.md`](../E2E_TESTS_README.md) for coverage, strategy, and why most
UI tests are skipped. **That document is the source of truth for what is tested**; this
file covers only how to run the suite. Test counts and per-test inventories deliberately
live in one place so the two cannot disagree.

## What is here

| Spec | Covers |
|------|--------|
| `alertQueries.spec.ts` | Alert rule pages with the Kinetica datasource |
| `configEditor.spec.ts` | Datasource configuration page |
| `dataQueries.spec.ts` | Provisioned dashboard loading |
| `queryEditor.spec.ts` | Query editor (currently all skipped, see the strategy doc) |
| `joinQueries.spec.ts` | Cross-schema joins in the query builder |
| `variableQueries.spec.ts` | Dashboard variables of type *Query* |

`joinQueries` and `variableQueries` need a live Kinetica behind the provisioned
datasource. They check the datasource health endpoint and skip themselves when it fails,
so CI (which runs Grafana without a database) is unaffected.

## Running the tests

Requires Node >= 22, Docker, and Docker Compose.

```bash
npm install
npx playwright install chromium
npm run build                      # the container serves the built plugin from dist/
docker compose up -d
curl http://localhost:3000/api/health
npm run e2e
```

To exercise the conditional tests as well, copy `.env.example` to `.env` and set your
Kinetica credentials before `docker compose up`.

## Environment

- **Grafana**: <http://localhost:3000>, `admin` / `admin`
- **Base URL**: taken from `GRAFANA_URL`, defaulting to `http://localhost:3000`
- **Browser**: Chromium, authenticated once and stored in `playwright/.auth/admin.json`
- **Plugin**: the repository root is mounted into the container's plugin directory and
  Grafana loads `dist/`. The mount is live, so a rebuild only needs a container restart.
- **Provisioning**: `provisioning/datasources/datasources.yml` and
  `provisioning/dashboards/` are mounted into the container and loaded at startup

## Writing tests

Prefer the patterns in the existing specs over the `@grafana/plugin-e2e` page-object
helpers. Those helpers resolve version-specific selectors internally, which is what made
most of the original suite unreliable across supported Grafana versions. Asserting on
provisioned dashboards, URL parameters, and the network requests the editor issues has
held up where selector-driven tests did not.

For a test that needs a live database, follow the health-check guard in
`variableQueries.spec.ts` so it skips rather than fails where none is available.

## Debugging

```bash
npx playwright test --ui                        # interactive runner
npx playwright test --debug                     # step through
npm run e2e -- --grep "populates a variable"    # single test
npx playwright show-report                      # last HTML report
```

## Troubleshooting

**Port already in use.** Change the published port in `docker-compose.yaml` and
`.config/docker-compose-base.yaml`, then point Playwright at it with `GRAFANA_URL`.

**Authentication failures.** Delete the stored session and re-run:

```bash
rm -rf playwright/.auth && npm run e2e
```

**Plugin not loading.** Rebuild with `npm run build`, restart with
`docker compose restart grafana`, then check `docker logs kinetica-grafana-dev`.

**Datasource not found, or queries failing.** Confirm `.env` exists with valid values,
then recreate the container so it is re-read: `docker compose up -d --force-recreate`.
Note that editing a provisioned datasource through the Grafana API bumps its version and
stops provisioning from updating it. Delete the datasource and restart to recover.
