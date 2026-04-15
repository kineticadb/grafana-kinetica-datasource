import { DataSourceInstanceSettings, CoreApp, DataQueryRequest, DataQueryResponse, DataFrame } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { KineticaDataSourceOptions, KineticaQuery, defaultQuery } from './types';

export class DataSource extends DataSourceWithBackend<KineticaQuery, KineticaDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<KineticaDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(app: CoreApp): Partial<KineticaQuery> {
    return defaultQuery;
  }

  // Intercept the query response to fix column ordering
  query(request: DataQueryRequest<KineticaQuery>): Observable<DataQueryResponse> {
    return super.query(request).pipe(
      map((response) => {
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

  // 1. Get Schemas
  async getSchemas(): Promise<string[]> {
    return this.getResource('schemas');
  }
  
  // 2. Get Tables
  async getTableNames(schema?: string): Promise<string[]> {
    return this.getResource('tables', { schema });
  }

  // 3. Get Columns
  async getColumns(schema: string | undefined, tableName: string): Promise<string[]> {
    const params: any = { table: tableName };
    if (schema) {
      params.schema = schema;
    }
    return this.getResource('columns', params);
  }
}
