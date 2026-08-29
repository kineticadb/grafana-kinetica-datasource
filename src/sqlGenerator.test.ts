import { generateSql, splitQualifiedName } from './sqlGenerator';
import { KineticaQueryBuilder } from './types';

const withJoin = (baseSchema: string, baseTable: string, joinSchema: string, joinTable: string): string =>
  generateSql({
    schema: baseSchema,
    table: baseTable,
    selects: [],
    joins: [{ type: 'INNER', schema: joinSchema, table: joinTable }],
  } as KineticaQueryBuilder);

describe('splitQualifiedName', () => {
  it('splits a qualified name at the first dot', () => {
    expect(splitQualifiedName('other.events')).toEqual({ schema: 'other', table: 'events' });
  });

  it('uses the fallback schema for a bare name', () => {
    expect(splitQualifiedName('events', 'prod')).toEqual({ schema: 'prod', table: 'events' });
  });

  it('prefers the name\'s own schema over the fallback', () => {
    expect(splitQualifiedName('other.events', 'prod')).toEqual({ schema: 'other', table: 'events' });
  });

  it('strips quotes', () => {
    expect(splitQualifiedName('"other"."events"')).toEqual({ schema: 'other', table: 'events' });
  });

  it('handles an empty name', () => {
    expect(splitQualifiedName('', 'prod')).toEqual({ schema: 'prod', table: '' });
  });
});

describe('generateSql JOIN qualification', () => {
  it('joins a table in another schema using that schema', () => {
    // Regression: the base schema used to be prefixed onto the qualified name,
    // producing "prod"."other.events" -- a table whose name contains a dot.
    expect(withJoin('prod', 'prod.orders', '', 'other.events')).toContain('INNER JOIN "other"."events"');
  });

  it('does not double-qualify a join table in the base schema', () => {
    expect(withJoin('prod', 'prod.orders', '', 'prod.events')).toContain('INNER JOIN "prod"."events"');
  });

  it('falls back to the base schema for an unqualified join table', () => {
    expect(withJoin('prod', 'prod.orders', '', 'events')).toContain('INNER JOIN "prod"."events"');
  });

  it('honours an explicit join schema for an unqualified table', () => {
    expect(withJoin('prod', 'prod.orders', 'other', 'events')).toContain('INNER JOIN "other"."events"');
  });

  it('lets the qualified name win over a conflicting explicit join schema', () => {
    // The name the user picked from the dropdown is the more specific signal.
    expect(withJoin('prod', 'prod.orders', 'stale', 'other.events')).toContain('INNER JOIN "other"."events"');
  });
});
