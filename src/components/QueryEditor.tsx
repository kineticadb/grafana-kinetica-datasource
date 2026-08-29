import React, { useState, useEffect, useMemo } from 'react';
import {
  Field,
  Combobox,
  MultiCombobox,
  Input,
  Button,
  IconButton,
  Switch,
  Alert,
  Stack,
  CodeEditor,
  useStyles2,
} from '@grafana/ui';
import { QueryEditorProps, GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { DataSource, resolveIdentifier, variableIdentifierOptions } from '../datasource';
import {
  KineticaQuery,
  KineticaDataSourceOptions,
  KineticaQueryBuilder,
  KineticaJoin,
  KineticaSetOperator,
  KineticaFilter,
  KineticaOrderBy,
  KineticaSelect,
  KineticaJoinCondition
} from '../types';
import { generateSql } from '../sqlGenerator';

type Props = QueryEditorProps<DataSource, KineticaQuery, KineticaDataSourceOptions>;

// Helper: Strip schema
const getSimpleTableName = (fullName: string) => {
    if (!fullName) {return '';}
    const parts = fullName.split('.');
    return parts.length > 1 ? parts[parts.length - 1] : fullName;
};

// Helper: Quote identifiers safely
const quoteIdentifier = (val: string) => {
    if (!val) { return ''; }
    const clean = val.replace(/"/g, '');
    return clean.split('.').map(p => `"${p}"`).join('.');
};

interface RichColumnOption {
    label: string;
    value: string;
    rawColumn: string;
    rawTableAlias: string;
    sourceIndex: number;
}

const getStyles = (theme: GrafanaTheme2) => ({
  builderContainer: css`
    border: 1px solid ${theme.colors.border.weak};
    padding: ${theme.spacing(2)};
    border-radius: ${theme.shape.radius.default};
  `,
  builderTitle: css`
    margin-top: 0;
    margin-bottom: ${theme.spacing(2.5)};
    color: ${theme.colors.text.link};
    border-bottom: 1px solid ${theme.colors.border.weak};
    padding-bottom: ${theme.spacing(0.5)};
  `,
  sectionTitle: css`
    font-size: 11px;
    color: ${theme.colors.text.secondary};
  `,
  selectRow: css`
    margin-bottom: ${theme.spacing(0.5)};
  `,
  joinContainer: css`
    margin-bottom: ${theme.spacing(2)};
    padding: ${theme.spacing(1.25)};
    background-color: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.radius.default};
  `,
  joinConditionsContainer: css`
    padding-left: ${theme.spacing(1.25)};
    margin-top: ${theme.spacing(0.5)};
  `,
  joinConditionsLabel: css`
    font-size: 11px;
    color: ${theme.colors.text.secondary};
    margin-bottom: ${theme.spacing(0.5)};
  `,
  joinConditionsEmpty: css`
    font-style: italic;
    font-size: 11px;
    color: ${theme.colors.text.disabled};
  `,
  filterRow: css`
    margin-bottom: ${theme.spacing(0.5)};
  `,
  groupSection: css`
    margin-bottom: ${theme.spacing(2)};
    padding: ${theme.spacing(1.25)} 0;
    border-top: 1px dashed ${theme.colors.border.medium};
    border-bottom: 1px dashed ${theme.colors.border.medium};
  `,
  havingSection: css`
    margin-bottom: ${theme.spacing(2)};
    margin-left: ${theme.spacing(1.25)};
  `,
  orderByRow: css`
    margin-bottom: ${theme.spacing(0.5)};
  `,
  setOpContainer: css`
    margin-top: ${theme.spacing(2.5)};
  `,
  setOpHeader: css`
    margin-bottom: ${theme.spacing(1.25)};
  `,
  setOpLabel: css`
    font-weight: ${theme.typography.fontWeightBold};
    color: ${theme.colors.text.link};
    align-self: center;
  `,
  setOpBuilder: css`
    background-color: ${theme.colors.background.secondary};
    padding: ${theme.spacing(1.25)};
    border-radius: ${theme.shape.radius.default};
  `,
  editorHeader: css`
    display: flex;
    justify-content: flex-end;
    margin-bottom: ${theme.spacing(1)};
    gap: ${theme.spacing(1.25)};
  `,
  rawSqlContainer: css`
    border: 1px solid ${theme.colors.border.medium};
    border-radius: ${theme.shape.radius.default};
  `,
  formGroup: css`
    margin-bottom: ${theme.spacing(2)};
  `,
  sqlPreview: css`
    margin-top: ${theme.spacing(2.5)};
    font-family: ${theme.typography.fontFamilyMonospace};
  `,
  logicSpacer: css`
    width: 90px;
  `,
  logicSpacerSmall: css`
    width: 80px;
  `,
  // Additional styles to replace inline styles
  alertMargin: css`
    margin-bottom: ${theme.spacing(1.25)};
  `,
  flexGrow: css`
    flex-grow: 1;
  `,
  sectionMargin: css`
    margin-bottom: ${theme.spacing(2)};
  `,
  fieldNoMargin: css`
    margin-bottom: 0;
  `,
  iconButtonTopMargin: css`
    margin-top: ${theme.spacing(2.75)};
  `,
  iconButtonNoMargin: css`
    margin-top: 0;
  `,
  joinOperator: css`
    font-weight: ${theme.typography.fontWeightBold};
  `,
  addConditionButton: css`
    margin-top: ${theme.spacing(0.5)};
  `,
  groupByRow: css`
    display: flex;
    gap: ${theme.spacing(1.25)};
    align-items: flex-end;
    margin-bottom: ${theme.spacing(2)};
  `,
  orderBySection: css`
    margin-top: ${theme.spacing(1.25)};
  `,
  setOpAddButton: css`
    margin-top: ${theme.spacing(1.25)};
  `,
});

// -----------------------------------------------------------------------------
// REUSABLE BUILDER COMPONENT
// -----------------------------------------------------------------------------
interface BuilderFormProps {
  datasource: DataSource;
  builder: KineticaQueryBuilder;
  onChange: (b: KineticaQueryBuilder) => void;
  isRoot?: boolean;
}

const BuilderForm: React.FC<BuilderFormProps> = ({ datasource, builder, onChange, isRoot }) => {
  const styles = useStyles2(getStyles);
  const [schemaOptions, setSchemaOptions] = useState<Array<{ label: string; value: string }>>([]);
  // Keyed by schema: a query can reference several (the base table plus joins against
  // other schemas), and one shared list would offer the base schema's tables everywhere.
  const [tablesBySchema, setTablesBySchema] = useState<Record<string, Array<{ label: string; value: string }>>>({});
  const [allColumnOptions, setAllColumnOptions] = useState<RichColumnOption[]>([]);
  const [fetchError, setFetchError] = useState<string | null>(null);

  // Resolved during render rather than memoised on the raw strings: a dashboard
  // variable's *value* can change while `builder.schema` stays "$schema", so anything
  // keyed on the raw text would never re-fetch and the dropdowns would go stale.
  // Recomputing every render keeps the effect dependencies below honest.
  const resolvedSchema = resolveIdentifier(builder.schema);
  const resolvedTable = resolveIdentifier(builder.table);
  // Every table the column dropdown needs, with schema/table already interpolated.
  // Aliases keep the *raw* name so the labels match the SQL the builder generates.
  const columnSources = [
    ...(builder.table
      ? [{
          schema: resolvedSchema.name,
          table: resolvedTable.name,
          alias: (builder.alias && builder.alias.trim() !== '') ? builder.alias : getSimpleTableName(builder.table),
          index: 0,
          raw: builder.table,
        }]
      : []),
    ...(builder.joins ?? []).flatMap((j, idx) => j.table
      ? [{
          schema: resolveIdentifier(j.schema || builder.schema).name,
          table: resolveIdentifier(j.table).name,
          alias: (j.alias && j.alias.trim() !== '') ? j.alias : getSimpleTableName(j.table),
          index: idx + 1,
          raw: j.table,
        }]
      : []),
  ];
  const columnSourceKey = JSON.stringify(columnSources.map((c) => [c.schema, c.table, c.alias]));
  // A join with no schema of its own inherits the base table's.
  const joinSchemas = (builder.joins ?? []).map((j) => resolveIdentifier(j.schema || builder.schema).name);
  const neededSchemas = Array.from(new Set([resolvedSchema.name, ...joinSchemas].filter(Boolean)));
  const neededSchemasKey = neededSchemas.join('|');
  const tableOptionsFor = (schema: string) => tablesBySchema[schema] ?? [];
  const unresolvedName = [builder.schema, builder.table, ...(builder.joins ?? []).map((j) => j.table)]
    .find((v) => v && resolveIdentifier(v).unresolved);
  // Derived rather than pushed into state from an effect: the condition is a pure
  // function of the current props, and setState inside an effect is disallowed here.
  const variableError = unresolvedName
    ? `Variable "${unresolvedName}" has no value, so tables and columns cannot be listed.`
    : null;

  // Dashboard variables are offered alongside the fetched names so they can be picked
  // rather than typed blind, and so a stored "$var" still renders instead of showing an
  // empty control. Grouping keeps them distinct from a real object of the same name.
  const variableOptions = variableIdentifierOptions();
  const schemaSelectOptions = [...variableOptions, ...schemaOptions.map((o) => ({ ...o, group: 'Schemas' }))];
  const tableSelectOptionsFor = (schema: string) =>
    [...variableOptions, ...tableOptionsFor(schema).map((o) => ({ ...o, group: 'Tables' }))];
  const tableSelectOptions = tableSelectOptionsFor(resolvedSchema.name);

  useEffect(() => {
    let isMounted = true;
    datasource.getSchemas()
      .then((result) => {
        if (isMounted) {
          setSchemaOptions(result.data.map((v) => ({ label: v, value: v })));
          if (result.error) {
            setFetchError(result.error);
          }
        }
      })
      .catch((err) => {
        if (isMounted) {
          setSchemaOptions([]);
          setFetchError(`Failed to fetch schemas: ${err instanceof Error ? err.message : String(err)}`);
        }
      });
    return () => { isMounted = false; };
  }, [datasource]);

  useEffect(() => {
    let isMounted = true;
    const schemas = neededSchemasKey ? neededSchemasKey.split('|') : [];
    if (schemas.length > 0) {
      Promise.all(
        schemas.map(async (schema) => {
          const result = await datasource.getTableNames(schema);
          const prefix = `${schema}.`;
          return {
            schema,
            error: result.error,
            options: result.data.map((v) => ({
              label: v.startsWith(prefix) ? v.substring(prefix.length) : v,
              value: v,
            })),
          };
        })
      )
        .then((results) => {
          if (isMounted) {
            setTablesBySchema(Object.fromEntries(results.map((r) => [r.schema, r.options])));
            const failed = results.find((r) => r.error);
            if (failed?.error) {
              setFetchError(failed.error);
            }
          }
        })
        .catch((err) => {
          if (isMounted) {
            setTablesBySchema({});
            setFetchError(`Failed to fetch tables: ${err instanceof Error ? err.message : String(err)}`);
          }
        });
    }
    return () => { isMounted = false; };
  }, [datasource, neededSchemasKey]);

  const aliasError = useMemo(() => {
      const aliases = new Set<string>();
      const getRef = (alias: string | undefined, table: string) => (alias && alias.trim() !== '') ? alias : getSimpleTableName(table);
      if (builder.table) {aliases.add(getRef(builder.alias, builder.table));}
      if (builder.joins) {
          for (const j of builder.joins) {
              if (j.table) {
                  const ref = getRef(j.alias, j.table);
                  if (aliases.has(ref)) {return `Duplicate Alias: "${ref}"`;}
                  aliases.add(ref);
              }
          }
      }
      return null;
  }, [builder.alias, builder.table, builder.joins]);

  useEffect(() => {
    let isMounted = true;
    const fetchAllColumns = async () => {
        if (unresolvedName) { return; }
        const sources = columnSources;
        if (sources.length === 0) { if (isMounted) {setAllColumnOptions([]);} return; }
        try {
            const results = await Promise.all(sources.map(async (src) => {
                const result = await datasource.getColumns(src.schema, src.table);
                if (result.error && isMounted) {
                    setFetchError(result.error);
                }
                return result.data.map(c => ({ label: `${src.alias}.${c}`, value: `${src.alias}.${c}`, rawColumn: c, rawTableAlias: src.alias, sourceIndex: src.index }));
            }));
            if (isMounted) {setAllColumnOptions(results.flat());}
        } catch (err) {
            if (isMounted) {
                setAllColumnOptions([]);
                setFetchError(`Failed to fetch columns: ${err instanceof Error ? err.message : String(err)}`);
            }
        }
    };
    fetchAllColumns();
    return () => { isMounted = false; };
    // columnSources is rebuilt every render; the serialised key is what actually changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [datasource, columnSourceKey, unresolvedName]);

  const combinedOptions = useMemo(() => {
    const aliasedRawColumns = new Set<string>();
    const aliasOpts = builder.selects?.filter(s => s.alias && s.alias.trim() !== '').map(s => {
        const rawKey = s.table ? `${s.table}.${s.column}` : s.column;
        aliasedRawColumns.add(rawKey);
        return { label: `${s.alias} => ${rawKey}`, value: s.alias! };
    }) || [];
    const rawOpts = allColumnOptions.filter(c => !aliasedRawColumns.has(c.value)).map(c => ({ label: c.label, value: c.value }));
    return [...aliasOpts, ...rawOpts];
  }, [builder.selects, allColumnOptions]);

  const update = (field: keyof KineticaQueryBuilder, val: any) => { onChange({ ...builder, [field]: val }); };
  const onSchemaChange = (v: string) => { setTablesBySchema({}); setAllColumnOptions([]); onChange({ ...builder, schema: v, table: '', selects: [], joins: [], groupBy: [], orderBy: [] }); };
  const onTableChange = (v: string) => { setAllColumnOptions([]); onChange({ ...builder, table: v, selects: [], filters: [] }); };

  const updateSelect = (i: number, field: keyof KineticaSelect, val: string | undefined) => {
      const list = [...(builder.selects || [])];
      if (field === 'column') {
          const matchedOpt = allColumnOptions.find(o => o.value === val);
          if (matchedOpt) {
              list[i] = { ...list[i], column: matchedOpt.rawColumn, table: matchedOpt.rawTableAlias };
          } else {
              list[i] = { ...list[i], column: val ?? '', table: undefined };
          }
      } else {
          list[i] = { ...list[i], [field]: val };
      }
      update('selects', list);
  };
  const addSelect = () => update('selects', [...(builder.selects || []), { column: '' }]);
  const removeSelect = (i: number) => { const list = [...(builder.selects || [])]; list.splice(i, 1); update('selects', list); };

  const moveSelect = (i: number, direction: -1 | 1) => {
      const list = [...(builder.selects || [])];
      if (i + direction < 0 || i + direction >= list.length) {return;}
      const temp = list[i];
      list[i] = list[i + direction];
      list[i + direction] = temp;
      update('selects', list);
  };

  const updateJoin = (i: number, field: keyof KineticaJoin, val: string | KineticaJoinCondition[] | undefined) => {
    const list = [...(builder.joins || [])];
    list[i] = { ...list[i], [field]: val };
    update('joins', list);
  };
  const addJoin = () => update('joins', [...(builder.joins || []), { type: 'INNER', schema: builder.schema, table: '', conditions: [] }]);
  const removeJoin = (i: number) => {
      const joins = [...(builder.joins || [])];
      const joinToRemove = joins[i];
      const removedAlias = joinToRemove.alias || getSimpleTableName(joinToRemove.table);
      joins.splice(i, 1);
      let newSelects = builder.selects;
      if (removedAlias && builder.selects) {
          const prefix = `${removedAlias}.`;
          newSelects = builder.selects.filter(s => {
              if (s.table === removedAlias) {return false;}
              if (!s.table && s.column && s.column.startsWith(prefix)) {return false;}
              return true;
          });
      }
      onChange({ ...builder, joins: joins, selects: newSelects });
  };

  const addJoinCondition = (joinIdx: number) => {
      const joins = [...(builder.joins || [])];
      const conditions = joins[joinIdx].conditions || [];
      conditions.push({ logic: 'AND', left: '', operator: '=', right: '' });
      joins[joinIdx] = { ...joins[joinIdx], conditions };
      update('joins', joins);
  };
  const removeJoinCondition = (joinIdx: number, condIdx: number) => {
      const joins = [...(builder.joins || [])];
      const conditions = [...(joins[joinIdx].conditions || [])];
      conditions.splice(condIdx, 1);
      joins[joinIdx] = { ...joins[joinIdx], conditions };
      update('joins', joins);
  };
  const updateJoinCondition = (joinIdx: number, condIdx: number, field: keyof KineticaJoinCondition, val: string) => {
      const joins = [...(builder.joins || [])];
      const conditions = [...(joins[joinIdx].conditions || [])];
      if (field === 'left' || field === 'right') {
          conditions[condIdx] = { ...conditions[condIdx], [field]: quoteIdentifier(val) };
      } else {
          conditions[condIdx] = { ...conditions[condIdx], [field]: val as 'AND' | 'OR' | string };
      }
      joins[joinIdx] = { ...joins[joinIdx], conditions };
      update('joins', joins);
  };

  const updateFilterList = (key: 'filters' | 'having', i: number, field: keyof KineticaFilter, val: any) => {
    const list = [...(builder[key] || [])];
    list[i] = { ...list[i], [field]: val };
    update(key, list);
  };
  const addFilterList = (key: 'filters' | 'having') => update(key, [...(builder[key] || []), { logic: 'AND', key: '', operator: '=', value: '' }]);
  const removeFilterList = (key: 'filters' | 'having', i: number) => { const list = [...(builder[key] || [])]; list.splice(i, 1); update(key, list); };

  const updateOrder = (i: number, field: keyof KineticaOrderBy, val: string | undefined) => {
    const list = [...(builder.orderBy || [])];
    list[i] = { ...list[i], [field]: val as string & ('ASC' | 'DESC') };
    update('orderBy', list);
  };
  const addOrder = () => update('orderBy', [...(builder.orderBy || []), { column: '', direction: 'ASC' }]);
  const removeOrder = (i: number) => { const list = [...(builder.orderBy || [])]; list.splice(i, 1); update('orderBy', list); };

  const addSetOp = () => update('setOperations', [...(builder.setOperations || []), { operator: 'UNION', query: { schema: builder.schema, table: '', selects: [], limit: 1000 } }]);
  const updateSetOp = (idx: number, newSub: KineticaQueryBuilder) => { const ops = [...(builder.setOperations || [])]; ops[idx].query = newSub; update('setOperations', ops); };
  const updateSetOperator = (idx: number, op: KineticaSetOperator) => { const ops = [...(builder.setOperations || [])]; ops[idx].operator = op; update('setOperations', ops); };
  const removeSetOp = (idx: number) => { const ops = [...(builder.setOperations || [])]; ops.splice(idx, 1); update('setOperations', ops); };

  const AGG_FUNCS = [{ label: '-- None --', value: '' }, ...['AVG', 'COUNT', 'MAX', 'MIN', 'SUM', 'VAR', 'STDDEV_POP', 'STDDEV_SAMP', 'VAR_POP', 'VAR_SAMP'].map(t => ({label: t, value: t}))];
  const JOIN_TYPES = ['INNER', 'LEFT', 'RIGHT', 'FULL'].map(t => ({ label: t, value: t }));
  const LOGIC_OPS = [{ label: 'AND', value: 'AND' }, { label: 'OR', value: 'OR' }];
  const SORT_DIRS = ['ASC', 'DESC'].map(t => ({ label: t, value: t }));
  const SET_OPS = ['UNION', 'UNION ALL', 'INTERSECT', 'INTERSECT ALL', 'EXCEPT', 'EXCEPT ALL'].map(t => ({ label: t, value: t }));

  const clearFetchError = () => setFetchError(null);

  return (
    <div className={styles.builderContainer}>
      {isRoot && <h5 className={styles.builderTitle}>SQL Builder</h5>}
      {(variableError ?? fetchError) && (
        <Alert title="Connection Error" severity="warning" className={styles.alertMargin} onRemove={clearFetchError}>
          {variableError ?? fetchError}
        </Alert>
      )}
      {aliasError && <Alert title="Validation Error" severity="error" className={styles.alertMargin}>{String(aliasError)}</Alert>}

      <Stack direction="row" gap={1}>
        <Field label="Schema"><Combobox options={schemaSelectOptions} value={builder.schema || null} onChange={v => onSchemaChange(v?.value ?? '')} width={25} createCustomValue /></Field>
        <div className={styles.flexGrow}><Field label="Table"><Combobox options={tableSelectOptions} value={builder.table || null} onChange={v => onTableChange(v?.value ?? '')} createCustomValue /></Field></div>
        <Field label="Alias"><Input value={builder.alias || ''} onChange={e => update('alias', e.currentTarget.value)} placeholder="Alias" width={10} /></Field>
        <Field label="Distinct"><Switch value={builder.distinct ?? false} onChange={e => update('distinct', e.currentTarget.checked)} /></Field>
      </Stack>

      {/* SELECT COLS */}
      <div className={styles.sectionMargin}>
        <h6 className={styles.sectionTitle}>Selected Columns</h6>
        {builder.selects?.map((s, i) => (
            <div key={i} className={styles.selectRow}>
                <Stack direction="row" gap={1}>
                    <Field label={i===0?"Aggregate":""} className={styles.fieldNoMargin}>
                        <Combobox options={AGG_FUNCS} value={s.aggregate || null} onChange={v => updateSelect(i, 'aggregate', (v?.value === '' || v?.value === undefined) ? undefined : v.value)} width={24} placeholder="Func" />
                    </Field>
                    <div className={styles.flexGrow}>
                         <Field label={i===0?"Column":""} className={styles.fieldNoMargin}>
                            <Combobox options={allColumnOptions} value={s.table ? `${s.table}.${s.column}` : (s.column || null)} onChange={v => updateSelect(i, 'column', v?.value)} />
                         </Field>
                    </div>
                    <Field label={i===0?"Alias":""} className={styles.fieldNoMargin}>
                        <Input value={s.alias || ''} onChange={e => updateSelect(i, 'alias', e.currentTarget.value)} width={16} placeholder="Alias" />
                    </Field>
                    <div className={i===0 ? styles.iconButtonTopMargin : styles.iconButtonNoMargin}>
                        <Stack direction="row" gap={0.5}>
                            <IconButton name="arrow-up" size="sm" variant="secondary" aria-label="Move Up" disabled={i===0} onClick={() => moveSelect(i, -1)} />
                            <IconButton name="arrow-down" size="sm" variant="secondary" aria-label="Move Down" disabled={i===(builder.selects?.length||0)-1} onClick={() => moveSelect(i, 1)} />
                            <IconButton name="trash-alt" variant="secondary" aria-label="Remove Column" onClick={() => removeSelect(i)} />
                        </Stack>
                    </div>
                </Stack>
            </div>
        ))}
        <Button size="sm" variant="secondary" icon="plus" onClick={addSelect}>Add Column</Button>
      </div>

      {/* JOINS */}
      {builder.joins?.map((j, i) => {
          const leftOptions = allColumnOptions.filter(c => c.sourceIndex <= i).map(c => ({ label: c.label, value: c.value }));
          const rightOptions = allColumnOptions.filter(c => c.sourceIndex === i + 1).map(c => ({ label: c.label, value: c.value }));

          return (
            <div key={i} className={styles.joinContainer}>
              <Stack direction="row" gap={1}>
                <Field label={i === 0 ? 'Type' : ''}><Combobox options={JOIN_TYPES} value={j.type || null} onChange={v => updateJoin(i, 'type', v?.value)} width={20} /></Field>
                <Field label={i === 0 ? 'Join Schema' : ''}><Combobox options={schemaSelectOptions} value={j.schema || null} onChange={v => updateJoin(i, 'schema', v?.value ?? '')} width={22} placeholder={resolvedSchema.name || 'Base schema'} createCustomValue /></Field>
                <div className={styles.flexGrow}><Field label={i === 0 ? 'Join Table' : ''}><Combobox options={tableSelectOptionsFor(joinSchemas[i])} value={j.table || null} onChange={v => updateJoin(i, 'table', v?.value)} createCustomValue /></Field></div>
                <Field label={i === 0 ? 'Alias' : ''}><Input value={j.alias || ''} onChange={e => updateJoin(i, 'alias', e.currentTarget.value)} placeholder="Alias" width={10} /></Field>
                <div className={i === 0 ? styles.iconButtonTopMargin : styles.iconButtonNoMargin}><IconButton name="trash-alt" variant="secondary" aria-label="Remove Join" onClick={() => removeJoin(i)} /></div>
              </Stack>

              <div className={styles.joinConditionsContainer}>
                  <div className={styles.joinConditionsLabel}>ON Conditions</div>
                  {(!j.conditions || j.conditions.length === 0) && (
                      <div className={styles.joinConditionsEmpty}>No conditions added.</div>
                  )}
                  {j.conditions?.map((cond, condIdx) => (
                      <Stack key={condIdx} direction="row" gap={1} alignItems="center">
                          {condIdx > 0 && (
                              <Combobox options={LOGIC_OPS} value={cond.logic || 'AND'} onChange={v => updateJoinCondition(i, condIdx, 'logic', v.value ?? 'AND')} width={16} />
                          )}
                          {condIdx === 0 && <div className={styles.logicSpacerSmall}></div>}

                          <Combobox options={leftOptions} placeholder="Left" value={cond.left || null} onChange={v => updateJoinCondition(i, condIdx, 'left', v?.value ?? '')} />
                          <div className={styles.joinOperator}>=</div>
                          <Combobox options={rightOptions} placeholder="Right" value={cond.right || null} onChange={v => updateJoinCondition(i, condIdx, 'right', v?.value ?? '')} />
                          <IconButton name="trash-alt" size="sm" variant="secondary" aria-label="Remove Condition" onClick={() => removeJoinCondition(i, condIdx)} />
                      </Stack>
                  ))}
                  <Button size="xs" variant="secondary" icon="plus" className={styles.addConditionButton} onClick={() => addJoinCondition(i)}>Add Condition</Button>
              </div>
            </div>
          );
      })}
      <div className={styles.sectionMargin}><Button size="sm" variant="secondary" icon="plus" onClick={addJoin}>Add Join</Button></div>

      {/* WHERE */}
      {builder.filters?.map((f, i) => (
        <div key={i} className={styles.filterRow}>
          <Stack direction="row" gap={1}>
            {i > 0 && <Combobox options={LOGIC_OPS} value={f.logic || 'AND'} onChange={v => updateFilterList('filters', i, 'logic', v.value)} width={16} />}
            {i === 0 && <div className={styles.logicSpacer}></div>}
            <Combobox options={allColumnOptions} value={f.key || null} onChange={v => updateFilterList('filters', i, 'key', v?.value)} width={20} />
            <Input value={f.operator || ''} onChange={e => updateFilterList('filters', i, 'operator', e.currentTarget.value)} placeholder="operator" width={12} />
            <div className={styles.flexGrow}><Input value={f.value || ''} onChange={e => updateFilterList('filters', i, 'value', e.currentTarget.value)} placeholder="value" /></div>
            <IconButton name="trash-alt" variant="secondary" aria-label="Remove Filter" onClick={() => removeFilterList('filters', i)} />
          </Stack>
        </div>
      ))}
      <div className={styles.sectionMargin}>
        <Stack direction="row" gap={2}>
          <Button size="sm" variant="secondary" icon="plus" onClick={() => addFilterList('filters')}>Add Filter</Button>
          <Field label="Time Column" className={styles.fieldNoMargin}><Combobox options={allColumnOptions} value={builder.timeColumn || null} onChange={v => update('timeColumn', v?.value)} width={20} /></Field>
        </Stack>
      </div>

      {/* GROUP / HAVING / ORDER / LIMIT */}
      <div className={styles.groupSection}>
        <div className={styles.groupByRow}>
           <div className={styles.flexGrow}>
             <Field label="Group By"><MultiCombobox options={combinedOptions} value={(builder.groupBy || []).map((c) => ({ label: c, value: c }))} onChange={v => update('groupBy', v.map((i) => i.value!))} /></Field>
           </div>
           <Field label="Limit"><Input type="number" value={builder.limit !== undefined ? builder.limit : 1000} onChange={e => update('limit', parseInt(e.currentTarget.value, 10) || 1000)} width={10} /></Field>
        </div>

        {builder.groupBy && builder.groupBy.length > 0 && (
          <div className={styles.havingSection}>
            <h6 className={styles.sectionTitle}>Having Clause</h6>
            {builder.having?.map((f, i) => (
              <div key={i} className={styles.filterRow}>
                <Stack direction="row" gap={1}>
                  {i > 0 && <Combobox options={LOGIC_OPS} value={f.logic || 'AND'} onChange={v => updateFilterList('having', i, 'logic', v.value)} width={16} />}
                  {i === 0 && <div className={styles.logicSpacer}></div>}
                  <Combobox options={combinedOptions} value={f.key || null} onChange={v => updateFilterList('having', i, 'key', v?.value)} width={20} />
                  <Input value={f.operator || ''} onChange={e => updateFilterList('having', i, 'operator', e.currentTarget.value)} placeholder="operator" width={12} />
                  <Input value={f.value || ''} onChange={e => updateFilterList('having', i, 'value', e.currentTarget.value)} className={styles.flexGrow} placeholder="value" />
                  <IconButton name="trash-alt" variant="secondary" aria-label="Remove Having" onClick={() => removeFilterList('having', i)} />
                </Stack>
              </div>
            ))}
            <Button size="sm" variant="secondary" icon="plus" onClick={() => addFilterList('having')}>Add Having</Button>
          </div>
        )}

        <div className={styles.orderBySection}>
          <h6 className={styles.sectionTitle}>Order By</h6>
          {builder.orderBy?.map((o, i) => (
             <div key={i} className={styles.orderByRow}>
               <Stack direction="row" gap={1}>
                 <Combobox options={combinedOptions} value={o.column || null} onChange={v => updateOrder(i, 'column', v?.value)} width={20} />
                 <Combobox options={SORT_DIRS} value={o.direction || 'ASC'} onChange={v => updateOrder(i, 'direction', v?.value)} width={16} />
                 <IconButton name="trash-alt" variant="secondary" aria-label="Remove Sort" onClick={() => removeOrder(i)} />
               </Stack>
             </div>
          ))}
          <Button size="sm" variant="secondary" icon="plus" onClick={addOrder}>Add Sort</Button>
        </div>
      </div>

      {/* SET OPS */}
      {builder.setOperations?.map((op, i) => (
        <div key={i} className={styles.setOpContainer}>
          <div className={styles.setOpHeader}>
            <Stack direction="row" gap={2}>
              <div className={styles.setOpLabel}>OPERATOR:</div>
              <Combobox options={SET_OPS} value={op.operator || 'UNION'} onChange={(v) => updateSetOperator(i, v?.value as KineticaSetOperator)} width={35} />
              <IconButton name="trash-alt" variant="secondary" aria-label="Remove Set Op" onClick={() => removeSetOp(i)} />
            </Stack>
          </div>
          <div className={styles.setOpBuilder}>
            <BuilderForm datasource={datasource} builder={op.query} onChange={(newSub) => updateSetOp(i, newSub)} isRoot={false} />
          </div>
        </div>
      ))}
      <div className={styles.setOpAddButton}>
        <Button variant="secondary" icon="plus" onClick={addSetOp}>Add Set Operation</Button>
      </div>
    </div>
  );
};

export function QueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  const styles = useStyles2(getStyles);

  // Ensure rawSql is always a string
  const safeRawSql = typeof query.rawSql === 'string' ? query.rawSql : '';

  const [isRawSql, setIsRawSql] = useState(!query.builder);
  const rawBuilder = query.builder || { schema: '', table: '' };
  let initialSelects = rawBuilder.selects;
  if (!initialSelects && rawBuilder.columns) { initialSelects = rawBuilder.columns.map(c => ({ column: c })); }

  const builder: KineticaQueryBuilder = { filters: [], joins: [], setOperations: [], groupBy: [], having: [], orderBy: [], limit: 1000, ...rawBuilder, selects: initialSelects || [] };

  const onBuilderChange = (newBuilder: KineticaQueryBuilder) => {
    const sql = generateSql(newBuilder);
    onChange({ ...query, rawSql: sql, builder: newBuilder });
  };

  const onReset = () => {
      const emptyBuilder: KineticaQueryBuilder = { schema: '', table: '', alias: '', selects: [], filters: [], joins: [], setOperations: [], groupBy: [], having: [], orderBy: [], limit: 1000 };
      onBuilderChange(emptyBuilder);
  };

  // --- LOCAL STATE FOR RAW EDITOR ---
  // Using derived state pattern (sync during render) to avoid useEffect lint errors.
  const [tempRawSql, setTempRawSql] = useState(safeRawSql);
  const [lastSyncedSql, setLastSyncedSql] = useState(safeRawSql);

  // If props have changed externally since last sync:
  if (safeRawSql !== lastSyncedSql) {
      // Check if the change is just a flattened version of our current state
      // (This happens when we blur/save: formatting is lost in prop, but we want to keep it in editor)
      const currentFlattened = tempRawSql.replace(/[\r\n]+/g, ' ');
      const isJustFlattened = safeRawSql === currentFlattened;

      if (!isJustFlattened) {
          // It's a real external change (Reset, or query switch), so overwrite local state
          setTempRawSql(safeRawSql);
      }

      // Update tracker so we don't sync again until next prop change
      setLastSyncedSql(safeRawSql);
  }

  const onRawSqlBlur = () => {
      // Flatten: Replace newlines with spaces to satisfy "stored as single line" requirement
      const flattened = tempRawSql.replace(/[\r\n]+/g, ' ');

      // Update global state and Run
      onChange({ ...query, rawSql: flattened });
      onRunQuery();
  };

  return (
    <div>
      <div className={styles.editorHeader}>
        {!isRawSql && <Button variant="secondary" size="sm" onClick={onReset}>Reset</Button>}
        <Field label="Raw SQL Mode" className={styles.fieldNoMargin}><Switch value={isRawSql} onChange={() => setIsRawSql(!isRawSql)} /></Field>
      </div>
      {isRawSql ? (
        <div className={styles.rawSqlContainer}>
           <CodeEditor
              value={tempRawSql}
              language="sql"
              onChange={val => setTempRawSql(val)}
              onBlur={onRawSqlBlur}
              height="200px"
              showLineNumbers={true}
              showMiniMap={false}
              monacoOptions={{ wordWrap: 'on', scrollBeyondLastLine: false }}
           />
        </div>
      ) : (
        <div className={styles.formGroup}>
          <BuilderForm datasource={datasource} builder={builder} onChange={onBuilderChange} isRoot={true} />
          <Alert title="SQL Preview" severity="info" className={styles.sqlPreview}>{safeRawSql}</Alert>
          <Button onClick={onRunQuery} variant="primary">Run Query</Button>
        </div>
      )}
    </div>
  );
}
