import { CustomVariableSupport, DataQueryRequest, DataQueryResponse } from '@grafana/data';
import { Observable, from } from 'rxjs';
import { map } from 'rxjs/operators';

import type { DataSource } from './datasource';
import { KineticaVariableQuery } from './types';
import { VariableQueryEditor } from './components/VariableQueryEditor';

/**
 * Registers the Kinetica query editor for dashboard variables of type *Query*.
 *
 * A thin adapter over `DataSource.metricFindQuery`, so there is a single
 * implementation of the SQL-to-variable-values path.
 *
 * The `MetricFindValue[]` is returned as-is rather than wrapped in a DataFrame:
 * Grafana detects that shape and uses it directly, preserving the text/value
 * split. Wrapping it in a frame loses the distinction — Grafana then falls back
 * to using the label as the value.
 */
export class KineticaVariableSupport extends CustomVariableSupport<DataSource, KineticaVariableQuery> {
  constructor(private readonly datasource: DataSource) {
    super();
  }

  editor = VariableQueryEditor;

  query(request: DataQueryRequest<KineticaVariableQuery>): Observable<DataQueryResponse> {
    const target = request.targets[0];

    return from(
      this.datasource.metricFindQuery(target, { range: request.range, scopedVars: request.scopedVars })
    ).pipe(map((values) => ({ data: values })));
  }
}
