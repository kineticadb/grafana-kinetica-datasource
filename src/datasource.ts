import { DataSourceInstanceSettings, CoreApp, DataQueryRequest, DataQueryResponse, DataFrame, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { KineticaDataSourceOptions, KineticaQuery, defaultQuery } from './types';
import { escapeStringValue } from './sqlGenerator';

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

export class DataSource extends DataSourceWithBackend<KineticaQuery, KineticaDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<KineticaDataSourceOptions>) {
    super(instanceSettings);
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
