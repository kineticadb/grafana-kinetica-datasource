import { ScopedVars } from '@grafana/data';

const replaceMock = jest.fn();

jest.mock('@grafana/runtime', () => ({
  // datasource.ts extends this at import time, so a stub is required.
  DataSourceWithBackend: class {
    constructor(_settings: unknown) {}
  },
  getTemplateSrv: () => ({ replace: replaceMock }),
}));

import { DataSource, interpolateSqlVariable } from './datasource';
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
