import {
  DataSourceInstanceSettings,
  CoreApp,
  DataQueryRequest,
  DataQueryResponse,
  DataFrame,
  ScopedVars,
  MetricFindValue,
  LegacyMetricFindQueryOptions,
  getDefaultTimeRange,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { Observable, lastValueFrom } from 'rxjs';
import { map } from 'rxjs/operators';

import { KineticaDataSourceOptions, KineticaQuery, KineticaVariableQuery, defaultQuery } from './types';
import { escapeStringValue } from './sqlGenerator';
import { KineticaVariableSupport } from './variables';

// Result type for resource fetches that includes error information
export interface ResourceFetchResult<T> {
  data: T;
  error?: string;
}

/**
 * Formats a template variable value for interpolation into SQL.
 *
 * Follows the convention used by Grafana's own SQL datasources:
 *  - single-value variables interpolate verbatim, so they can be used as
 *    identifiers (`FROM $table`) or inside quotes the user wrote (`= '$name'`);
 *  - multi-value / "include all" variables are quoted and comma-joined so they
 *    work in `IN ($hosts)`.
 * Without this, a multi-value variable interpolates as `{a,b}`, which is not
 * valid SQL.
 */
export function interpolateSqlVariable(value: unknown, variable?: { multi?: boolean; includeAll?: boolean }): unknown {
  if (typeof value === 'number' || typeof value === 'boolean') {
    return value;
  }
  if (Array.isArray(value)) {
    return value.map((v) => `'${escapeStringValue(String(v))}'`).join(',');
  }
  if (typeof value === 'string') {
    if (variable?.multi || variable?.includeAll) {
      return `'${escapeStringValue(value)}'`;
    }
    return value;
  }
  return value;
}

// refId used for the synthetic request behind a variable query. Variable queries
// have no panel target of their own, but the backend still keys results by refId.
const VARIABLE_REF_ID = 'metricFindQuery';

// Grafana's backendSrv cancels an in-flight request as soon as another one is
// issued with the same requestId. A dashboard resolves its variable queries
// concurrently, so a constant requestId makes each variable cancel the previous
// one and only the last survives. Counter guarantees uniqueness when the caller
// does not identify the variable.
let variableRequestSeq = 0;

function metricFindText(value: unknown): string {
  return value === null || value === undefined ? '' : String(value);
}

/**
 * Converts the first frame of a variable query into Grafana's MetricFindValue list.
 *
 * Follows the column convention used by Grafana's own SQL datasources: when the
 * result carries both a `__text` and a `__value` column they become the displayed
 * label and the stored value; otherwise the first column supplies both. Both are
 * required for the pair to apply, because a lone `__text` would silently discard
 * the value the user meant the variable to hold.
 *
 * Note the backend sorts columns alphabetically in `parseToFrame`, so these are
 * looked up by name rather than by position.
 */
export function frameToMetricFindValues(frame?: DataFrame): MetricFindValue[] {
  const fields = frame?.fields;
  if (!fields || fields.length === 0) {
    return [];
  }

  const textField = fields.find((f) => f.name === '__text');
  const valueField = fields.find((f) => f.name === '__value');

  if (textField && valueField) {
    const count = Math.min(textField.values.length, valueField.values.length);
    const values: MetricFindValue[] = [];
    for (let i = 0; i < count; i++) {
      const rawValue = valueField.values[i];
      values.push({
        text: metricFindText(textField.values[i]),
        // Numbers are kept as numbers so numeric variables compare correctly.
        value: typeof rawValue === 'number' ? rawValue : metricFindText(rawValue),
      });
    }
    return values;
  }

  return (fields[0].values ?? []).map((value: unknown) => ({ text: metricFindText(value) }));
}

export class DataSource extends DataSourceWithBackend<KineticaQuery, KineticaDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<KineticaDataSourceOptions>) {
    super(instanceSettings);
    // Registers the variable query editor; replaces the deprecated
    // DataSourcePlugin.setVariableQueryEditor().
    this.variables = new KineticaVariableSupport(this);
  }

  getDefaultQuery(app: CoreApp): Partial<KineticaQuery> {
    return defaultQuery;
  }

  /**
   * Interpolates dashboard/template variables before the query reaches the backend.
   *
   * `rawSql` is what the backend actually executes. The `builder` fields are also
   * interpolated because the backend reads `builder.schema` / `builder.table` as
   * metadata to look up column types (time-column detection); leaving a variable
   * uninterpolated there yields correct SQL but a failed metadata lookup, so time
   * columns would silently render as numbers.
   */
  applyTemplateVariables(query: KineticaQuery, scopedVars: ScopedVars): KineticaQuery {
    const templateSrv = getTemplateSrv();
    const interpolated: KineticaQuery = {
      ...query,
      rawSql: templateSrv.replace(query.rawSql ?? '', scopedVars, interpolateSqlVariable),
    };

    if (query.builder) {
      interpolated.builder = {
        ...query.builder,
        // No custom formatter: these are identifiers, so they must not be quoted.
        schema: templateSrv.replace(query.builder.schema ?? '', scopedVars),
        table: templateSrv.replace(query.builder.table ?? '', scopedVars),
      };
    }

    return interpolated;
  }

  /**
   * Populates a dashboard variable of type *Query* from a Kinetica query.
   *
   * The SQL is run through the normal query path rather than a bespoke request, so
   * backend macros still expand and `applyTemplateVariables` still runs — the latter
   * is what makes chained variables (one variable query referencing another) work.
   *
   * Accepts a bare string as well as a `KineticaVariableQuery` so that variables
   * saved before the query editor existed, and provisioned dashboards that store the
   * query as a string, keep working.
   */
  async metricFindQuery(
    query: Partial<KineticaVariableQuery> | string,
    options?: LegacyMetricFindQueryOptions
  ): Promise<MetricFindValue[]> {
    const rawSql = (typeof query === 'string' ? query : query?.rawSql) ?? '';
    if (!rawSql.trim()) {
      return [];
    }

    const request = {
      requestId: `${this.uid}-${VARIABLE_REF_ID}-${options?.variable?.name ?? `q${++variableRequestSeq}`}`,
      interval: '',
      intervalMs: 0,
      range: options?.range ?? getDefaultTimeRange(),
      scopedVars: options?.scopedVars ?? {},
      targets: [{ refId: VARIABLE_REF_ID, rawSql }],
      timezone: 'browser',
      app: CoreApp.Dashboard,
      startTime: Date.now(),
    } as DataQueryRequest<KineticaQuery>;

    const response = await lastValueFrom(this.query(request));
    return frameToMetricFindValues(response?.data?.[0] as DataFrame | undefined);
  }

  // Intercept the query response to fix column ordering
  query(request: DataQueryRequest<KineticaQuery>): Observable<DataQueryResponse> {
    return super.query(request).pipe(
      map((response) => {
        // Ensure response.data is an array
        if (!response || !Array.isArray(response.data)) {
          return response;
        }
        // Iterate over each DataFrame in the response (usually one per query)
        response.data.forEach((dataItem: any) => {
          // Ensure it's a valid DataFrame with fields
          if (!dataItem || !Array.isArray(dataItem.fields)) {
            return;
          }
          const frame = dataItem as DataFrame;

          // Find the original query object that generated this frame (match by refId)
          const query = request.targets.find((t) => t.refId === frame.refId);

          // Only reorder if we have Builder metadata to guide us
          if (query && query.builder && query.builder.selects && query.builder.selects.length > 0) {
            const selects = query.builder.selects;
            const orderedFields: any[] = [];
            const usedFieldIndices = new Set<number>();

            // 1. Iterate through the User's selected order
            selects.forEach((select) => {
              // Determine the expected field name (Alias takes priority, then Column)
              // We strip quotes just in case the backend returns clean names while frontend has "Name"
              const targetAlias = select.alias ? select.alias.replace(/"/g, '') : null;
              const targetCol = select.column ? select.column.replace(/"/g, '') : '';
              
              // Handle "table.col" -> just "col" if backend strips prefixes
              const shortCol = targetCol.includes('.') ? targetCol.split('.')[1] : targetCol;

              // Find the matching field in the DataFrame
              const fieldIndex = frame.fields.findIndex((f, idx) => {
                if (usedFieldIndices.has(idx)) { return false; } // Already picked
                
                const fieldName = f.name;
                
                // Match Logic:
                // A. Exact Alias Match
                if (targetAlias && fieldName === targetAlias) { return true; }
                // B. Exact Column Match
                if (fieldName === targetCol) { return true; }
                // C. Short Column Match (e.g. builder: t1.Open, frame: Open)
                if (fieldName === shortCol) { return true; }
                
                return false;
              });

              // If found, add to ordered list
              if (fieldIndex !== -1) {
                orderedFields.push(frame.fields[fieldIndex]);
                usedFieldIndices.add(fieldIndex);
              }
            });

            // 2. Append any remaining fields (e.g. Time column added implicitly, or unmapped fields)
            frame.fields.forEach((f, idx) => {
              if (!usedFieldIndices.has(idx)) {
                orderedFields.push(f);
              }
            });

            // 3. Apply the sorted fields back to the frame
            frame.fields = orderedFields;
          }
        });

        return response;
      })
    );
  }

  // Helper to extract error message from various error types
  private extractErrorMessage(err: unknown): string {
    if (err instanceof Error) {
      return err.message;
    }
    if (typeof err === 'object' && err !== null) {
      const errObj = err as Record<string, unknown>;
      if (typeof errObj.message === 'string') {
        return errObj.message;
      }
      if (typeof errObj.statusText === 'string') {
        return errObj.statusText;
      }
      if (typeof errObj.data === 'object' && errObj.data !== null) {
        const data = errObj.data as Record<string, unknown>;
        if (typeof data.message === 'string') {
          return data.message;
        }
      }
    }
    return String(err);
  }

  // 1. Get Schemas
  async getSchemas(): Promise<ResourceFetchResult<string[]>> {
    try {
      const result = await this.getResource('schemas');
      return { data: Array.isArray(result) ? result : [] };
    } catch (err) {
      const errorMessage = this.extractErrorMessage(err);
      return { data: [], error: `Failed to fetch schemas: ${errorMessage}` };
    }
  }

  // 2. Get Tables
  async getTableNames(schema?: string): Promise<ResourceFetchResult<string[]>> {
    try {
      const result = await this.getResource('tables', { schema });
      return { data: Array.isArray(result) ? result : [] };
    } catch (err) {
      const errorMessage = this.extractErrorMessage(err);
      return { data: [], error: `Failed to fetch tables: ${errorMessage}` };
    }
  }

  // 3. Get Columns
  async getColumns(schema: string | undefined, tableName: string): Promise<ResourceFetchResult<string[]>> {
    try {
      const params: Record<string, string> = { table: tableName };
      if (schema) {
        params.schema = schema;
      }
      const result = await this.getResource('columns', params);
      return { data: Array.isArray(result) ? result : [] };
    } catch (err) {
      const errorMessage = this.extractErrorMessage(err);
      return { data: [], error: `Failed to fetch columns: ${errorMessage}` };
    }
  }
}
