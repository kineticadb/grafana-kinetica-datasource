import { DataFrame, ScopedVars, TimeRange } from '@grafana/data';
import { of } from 'rxjs';

const replaceMock = jest.fn();
const getVariablesMock = jest.fn(() => [] as Array<{ name: string }>);

jest.mock('@grafana/runtime', () => ({
  // datasource.ts extends this at import time, so a stub is required.
  DataSourceWithBackend: class {
    constructor(_settings: unknown) {}
  },
  getTemplateSrv: () => ({ replace: replaceMock, getVariables: getVariablesMock }),
}));

import {
  DataSource,
  interpolateSqlVariable,
  frameToMetricFindValues,
  resolveIdentifier,
  variableIdentifierOptions,
} from './datasource';
import { KineticaQuery } from './types';

// Build an instance without running DataSourceWithBackend's constructor.
const ds = Object.create(DataSource.prototype) as DataSource;

describe('interpolateSqlVariable', () => {
  it('leaves a single-value string unquoted so it can be used as an identifier', () => {
    expect(interpolateSqlVariable('my_table', { multi: false })).toBe('my_table');
  });

  it('quotes a single-value string when the variable is multi-value', () => {
    expect(interpolateSqlVariable('web-01', { multi: true })).toBe("'web-01'");
  });

  it('quotes a single-value string when the variable includes All', () => {
    expect(interpolateSqlVariable('web-01', { includeAll: true })).toBe("'web-01'");
  });

  it('quotes and comma-joins arrays for use in IN (...)', () => {
    expect(interpolateSqlVariable(['a', 'b', 'c'])).toBe("'a','b','c'");
  });

  it('escapes embedded single quotes', () => {
    expect(interpolateSqlVariable("O'Brien", { multi: true })).toBe("'O''Brien'");
    expect(interpolateSqlVariable(["O'Brien"])).toBe("'O''Brien'");
  });

  it('passes numbers and booleans through unquoted', () => {
    expect(interpolateSqlVariable(42)).toBe(42);
    expect(interpolateSqlVariable(true)).toBe(true);
  });
});

describe('DataSource.applyTemplateVariables', () => {
  beforeEach(() => {
    replaceMock.mockReset();
    // Default: echo the input so assertions can focus on what gets passed in.
    replaceMock.mockImplementation((value: string) => value);
  });

  it('interpolates rawSql using the SQL formatter', () => {
    const query: KineticaQuery = { refId: 'A', rawSql: 'SELECT * FROM t WHERE h IN ($hosts)' };
    const scopedVars: ScopedVars = {};

    ds.applyTemplateVariables(query, scopedVars);

    expect(replaceMock).toHaveBeenCalledWith(
      'SELECT * FROM t WHERE h IN ($hosts)',
      scopedVars,
      interpolateSqlVariable
    );
  });

  it('interpolates builder.schema and builder.table without the quoting formatter', () => {
    const query: KineticaQuery = {
      refId: 'A',
      rawSql: 'SELECT 1',
      builder: { schema: '$schema', table: '$table' },
    };

    ds.applyTemplateVariables(query, {});

    // Identifiers must not be quoted, so no custom formatter is passed.
    expect(replaceMock).toHaveBeenCalledWith('$schema', {});
    expect(replaceMock).toHaveBeenCalledWith('$table', {});
  });

  it('returns the interpolated values in the resulting query', () => {
    replaceMock.mockImplementation((value: string) =>
      value.replace('$table', 'sensors').replace('$schema', 'prod')
    );

    const query: KineticaQuery = {
      refId: 'A',
      rawSql: 'SELECT * FROM $table',
      builder: { schema: '$schema', table: '$table' },
    };

    const result = ds.applyTemplateVariables(query, {});

    expect(result.rawSql).toBe('SELECT * FROM sensors');
    expect(result.builder?.schema).toBe('prod');
    expect(result.builder?.table).toBe('sensors');
  });

  it('leaves the query without a builder untouched', () => {
    const query: KineticaQuery = { refId: 'A', rawSql: 'SELECT 1' };
    const result = ds.applyTemplateVariables(query, {});
    expect(result.builder).toBeUndefined();
  });

  it('does not mutate the original query', () => {
    replaceMock.mockImplementation(() => 'CHANGED');
    const query: KineticaQuery = { refId: 'A', rawSql: 'SELECT 1', builder: { schema: 's', table: 't' } };

    ds.applyTemplateVariables(query, {});

    expect(query.rawSql).toBe('SELECT 1');
    expect(query.builder?.table).toBe('t');
  });

  it('tolerates a missing rawSql', () => {
    const query = { refId: 'A' } as KineticaQuery;
    expect(() => ds.applyTemplateVariables(query, {})).not.toThrow();
    expect(replaceMock).toHaveBeenCalledWith('', {}, interpolateSqlVariable);
  });
});

// Minimal stand-in for a backend frame; only `fields` matters to the conversion.
const frameWith = (fields: Array<{ name: string; values: unknown[] }>): DataFrame =>
  ({ fields, length: fields[0]?.values.length ?? 0 } as unknown as DataFrame);

describe('frameToMetricFindValues', () => {
  it('returns an empty list when there is no frame or no fields', () => {
    expect(frameToMetricFindValues(undefined)).toEqual([]);
    expect(frameToMetricFindValues(frameWith([]))).toEqual([]);
  });

  it('uses the first column when there is no __text / __value pair', () => {
    const frame = frameWith([
      { name: 'host', values: ['web-01', 'web-02'] },
      { name: 'ignored', values: [1, 2] },
    ]);
    expect(frameToMetricFindValues(frame)).toEqual([{ text: 'web-01' }, { text: 'web-02' }]);
  });

  it('maps __text to the label and __value to the stored value', () => {
    // Deliberately out of order: the backend sorts columns alphabetically, so the
    // pair must be found by name rather than by position.
    const frame = frameWith([
      { name: '__value', values: [10, 20] },
      { name: '__text', values: ['Ten', 'Twenty'] },
    ]);
    expect(frameToMetricFindValues(frame)).toEqual([
      { text: 'Ten', value: 10 },
      { text: 'Twenty', value: 20 },
    ]);
  });

  it('stringifies a non-numeric __value but leaves numbers as numbers', () => {
    const frame = frameWith([
      { name: '__text', values: ['A'] },
      { name: '__value', values: ['a-id'] },
    ]);
    expect(frameToMetricFindValues(frame)).toEqual([{ text: 'A', value: 'a-id' }]);
  });

  it('ignores __text when __value is absent, rather than dropping the value', () => {
    const frame = frameWith([{ name: '__text', values: ['only'] }]);
    expect(frameToMetricFindValues(frame)).toEqual([{ text: 'only' }]);
  });

  it('renders null and undefined as empty strings', () => {
    const frame = frameWith([{ name: 'host', values: ['web-01', null, undefined] }]);
    expect(frameToMetricFindValues(frame)).toEqual([{ text: 'web-01' }, { text: '' }, { text: '' }]);
  });

  it('stops at the shorter column when __text and __value differ in length', () => {
    const frame = frameWith([
      { name: '__text', values: ['A', 'B'] },
      { name: '__value', values: [1] },
    ]);
    expect(frameToMetricFindValues(frame)).toEqual([{ text: 'A', value: 1 }]);
  });
});

describe('DataSource.metricFindQuery', () => {
  let querySpy: jest.SpyInstance;

  beforeEach(() => {
    querySpy = jest
      .spyOn(DataSource.prototype, 'query')
      .mockReturnValue(of({ data: [frameWith([{ name: 'host', values: ['web-01', 'web-02'] }])] }));
  });

  afterEach(() => {
    querySpy.mockRestore();
  });

  it('returns an empty list without hitting the backend when the SQL is blank', async () => {
    await expect(ds.metricFindQuery('   ')).resolves.toEqual([]);
    await expect(ds.metricFindQuery({ rawSql: '' })).resolves.toEqual([]);
    expect(querySpy).not.toHaveBeenCalled();
  });

  it('accepts a bare string query', async () => {
    const result = await ds.metricFindQuery('SELECT host FROM hosts');

    expect(result).toEqual([{ text: 'web-01' }, { text: 'web-02' }]);
    expect(querySpy.mock.calls[0][0].targets[0].rawSql).toBe('SELECT host FROM hosts');
  });

  it('accepts a KineticaVariableQuery object', async () => {
    const result = await ds.metricFindQuery({ rawSql: 'SELECT host FROM hosts' });

    expect(result).toEqual([{ text: 'web-01' }, { text: 'web-02' }]);
    expect(querySpy.mock.calls[0][0].targets[0].rawSql).toBe('SELECT host FROM hosts');
  });

  it('passes the supplied range and scopedVars through, so macros and chaining work', async () => {
    const range = { from: 'from', to: 'to' } as unknown as TimeRange;
    const scopedVars: ScopedVars = { host: { text: 'web-01', value: 'web-01' } };

    await ds.metricFindQuery({ rawSql: 'SELECT 1' }, { range, scopedVars });

    const request = querySpy.mock.calls[0][0];
    expect(request.range).toBe(range);
    expect(request.scopedVars).toBe(scopedVars);
  });

  it('falls back to a default time range when the caller supplies none', async () => {
    await ds.metricFindQuery({ rawSql: 'SELECT 1' });
    expect(querySpy.mock.calls[0][0].range).toBeDefined();
  });

  it('gives each variable query a distinct requestId so they do not cancel each other', async () => {
    // Grafana's backendSrv aborts an in-flight request when another arrives with the
    // same requestId. A dashboard resolves its variables concurrently, so a shared id
    // meant only the last variable survived.
    await ds.metricFindQuery({ rawSql: 'SELECT a FROM t' });
    await ds.metricFindQuery({ rawSql: 'SELECT b FROM t' });

    const [first, second] = querySpy.mock.calls.map((c) => c[0].requestId);
    expect(first).not.toEqual(second);
  });

  it('uses the variable name in the requestId when the caller supplies one', async () => {
    await ds.metricFindQuery({ rawSql: 'SELECT a FROM t' }, { variable: { name: 'region' } });

    expect(querySpy.mock.calls[0][0].requestId).toContain('region');
  });

  it('returns an empty list when the response carries no frames', async () => {
    querySpy.mockReturnValue(of({ data: [] }));
    await expect(ds.metricFindQuery({ rawSql: 'SELECT 1' })).resolves.toEqual([]);
  });
});

describe('resolveIdentifier', () => {
  beforeEach(() => {
    replaceMock.mockReset();
  });

  it('returns an empty name for a blank value without calling the template service', () => {
    expect(resolveIdentifier(undefined)).toEqual({ name: '', unresolved: false });
    expect(resolveIdentifier('   ')).toEqual({ name: '', unresolved: false });
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it('passes a literal identifier through unchanged', () => {
    replaceMock.mockImplementation((v: string) => v);
    expect(resolveIdentifier('prod')).toEqual({ name: 'prod', unresolved: false });
  });

  it('interpolates a variable to its value', () => {
    replaceMock.mockImplementation(() => 'prod');
    expect(resolveIdentifier('$schema')).toEqual({ name: 'prod', unresolved: false });
  });

  it('takes the first entry of a multi-value variable', () => {
    // A multi-value variable interpolates as {a,b}, which cannot name a table.
    replaceMock.mockImplementation(() => '{prod,staging}');
    expect(resolveIdentifier('$schema')).toEqual({ name: 'prod', unresolved: false });
  });

  it('flags a variable that did not resolve', () => {
    // Grafana leaves an unknown variable in place rather than erroring.
    replaceMock.mockImplementation((v: string) => v);
    expect(resolveIdentifier('$nope')).toEqual({ name: '$nope', unresolved: true });
  });

  it('trims surrounding whitespace before interpolating', () => {
    replaceMock.mockImplementation((v: string) => v);
    expect(resolveIdentifier('  prod  ').name).toBe('prod');
    expect(replaceMock).toHaveBeenCalledWith('prod');
  });
});

describe('variableIdentifierOptions', () => {
  beforeEach(() => {
    replaceMock.mockReset();
    getVariablesMock.mockReset();
  });

  it('returns nothing when the dashboard has no variables', () => {
    getVariablesMock.mockReturnValue([]);
    expect(variableIdentifierOptions()).toEqual([]);
  });

  it('offers each variable as a pickable option that stores the reference, not the value', () => {
    getVariablesMock.mockReturnValue([{ name: 'schema' }, { name: 'tbl' }]);
    replaceMock.mockImplementation((v: string) => (v === '$schema' ? 'prod' : 'events'));

    expect(variableIdentifierOptions()).toEqual([
      { label: '$schema', value: '$schema', group: 'Variables', description: 'currently prod' },
      { label: '$tbl', value: '$tbl', group: 'Variables', description: 'currently events' },
    ]);
  });

  it('reports a variable with no current value rather than showing a bare $name', () => {
    getVariablesMock.mockReturnValue([{ name: 'nope' }]);
    replaceMock.mockImplementation((v: string) => v);

    expect(variableIdentifierOptions()[0].description).toBe('no current value');
  });

  it('describes a multi-value variable by its first entry, matching what a lookup would use', () => {
    getVariablesMock.mockReturnValue([{ name: 'schema' }]);
    replaceMock.mockImplementation(() => '{prod,staging}');

    expect(variableIdentifierOptions()[0].description).toBe('currently prod');
  });
});

describe('DataSource.getColumns', () => {
  let getResource: jest.Mock;

  beforeEach(() => {
    getResource = jest.fn().mockResolvedValue([]);
    (ds as unknown as { getResource: jest.Mock }).getResource = getResource;
  });

  it('uses the schema carried by a qualified table name', async () => {
    // Regression: the passed-in schema used to be prefixed onto the qualified name,
    // so a join against another schema looked up "prod.other.events".
    await ds.getColumns('prod', 'other.events');
    expect(getResource).toHaveBeenCalledWith('columns', { table: 'events', schema: 'other' });
  });

  it('falls back to the passed-in schema for a bare table name', async () => {
    await ds.getColumns('prod', 'events');
    expect(getResource).toHaveBeenCalledWith('columns', { table: 'events', schema: 'prod' });
  });

  it('omits the schema when neither the name nor the caller supplies one', async () => {
    await ds.getColumns(undefined, 'events');
    expect(getResource).toHaveBeenCalledWith('columns', { table: 'events' });
  });
});
