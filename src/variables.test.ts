import { DataQueryRequest, TimeRange, ScopedVars } from '@grafana/data';
import { lastValueFrom } from 'rxjs';

jest.mock('@grafana/runtime', () => ({
  DataSourceWithBackend: class {
    constructor(_settings: unknown) {}
  },
  getTemplateSrv: () => ({ replace: (v: string) => v }),
}));

// The editor pulls in Monaco, which is irrelevant to the query path under test.
jest.mock('./components/VariableQueryEditor', () => ({ VariableQueryEditor: () => null }));

import { KineticaVariableSupport } from './variables';
import { DataSource } from './datasource';
import { KineticaVariableQuery } from './types';

const metricFindQuery = jest.fn();
const datasource = { metricFindQuery } as unknown as DataSource;

const requestFor = (
  rawSql: string,
  extra: Partial<DataQueryRequest<KineticaVariableQuery>> = {}
): DataQueryRequest<KineticaVariableQuery> =>
  ({
    targets: [{ refId: 'A', rawSql }],
    range: { from: 'now-1h', to: 'now' } as unknown as TimeRange,
    scopedVars: {} as ScopedVars,
    ...extra,
  } as DataQueryRequest<KineticaVariableQuery>);

describe('KineticaVariableSupport', () => {
  beforeEach(() => {
    metricFindQuery.mockReset();
  });

  it('delegates to metricFindQuery, passing the target, range and scopedVars', async () => {
    metricFindQuery.mockResolvedValue([]);
    const range = { from: 'from', to: 'to' } as unknown as TimeRange;
    const scopedVars: ScopedVars = { region: { text: 'eu', value: 'eu' } };

    await lastValueFrom(new KineticaVariableSupport(datasource).query(requestFor('SELECT 1', { range, scopedVars })));

    expect(metricFindQuery).toHaveBeenCalledWith({ refId: 'A', rawSql: 'SELECT 1' }, { range, scopedVars });
  });

  it('returns the MetricFindValue list unwrapped', () => {
    // Deliberately NOT wrapped in a DataFrame: Grafana detects this shape and uses
    // it directly. Wrapping it collapses text/value, and the label ends up stored
    // as the variable's value.
    metricFindQuery.mockResolvedValue([{ text: 'web-01' }, { text: 'web-02' }]);

    return lastValueFrom(new KineticaVariableSupport(datasource).query(requestFor('SELECT host'))).then((res) => {
      expect(res.data).toEqual([{ text: 'web-01' }, { text: 'web-02' }]);
    });
  });

  it('keeps text and value distinct when the query returned both', async () => {
    metricFindQuery.mockResolvedValue([
      { text: 'Ten', value: 10 },
      { text: 'Twenty', value: 20 },
    ]);

    const response = await lastValueFrom(new KineticaVariableSupport(datasource).query(requestFor('SELECT 1')));

    expect(response.data).toEqual([
      { text: 'Ten', value: 10 },
      { text: 'Twenty', value: 20 },
    ]);
  });

  it('returns no frames when the query produced no values', async () => {
    metricFindQuery.mockResolvedValue([]);

    const response = await lastValueFrom(new KineticaVariableSupport(datasource).query(requestFor('SELECT 1')));

    expect(response.data).toEqual([]);
  });
});
