import React, { useState } from 'react';
import { CodeEditor, Field } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';

// Type-only: keeps the datasource -> variables -> editor import cycle out of runtime.
import type { DataSource } from '../datasource';
import { KineticaDataSourceOptions, KineticaQuery, KineticaVariableQuery } from '../types';

export type VariableQueryEditorProps = QueryEditorProps<
  DataSource,
  KineticaQuery,
  KineticaDataSourceOptions,
  KineticaVariableQuery
>;

/**
 * Editor for a dashboard variable of type *Query*.
 *
 * Committed on blur rather than on every keystroke, matching the raw SQL editor in
 * QueryEditor: each commit re-runs the variable query, so debouncing it to blur
 * avoids a query per character.
 */
export function VariableQueryEditor({ query, onChange, onRunQuery }: VariableQueryEditorProps) {
  // Variables saved before this editor existed, and provisioned dashboards that store
  // the query as plain SQL, arrive as a string rather than an object.
  const initialSql = (typeof query === 'string' ? query : query?.rawSql) ?? '';
  const [rawSql, setRawSql] = useState(initialSql);

  const onBlur = (value: string) => {
    onChange({ ...(typeof query === 'string' ? {} : query), rawSql: value } as KineticaVariableQuery);
    onRunQuery();
  };

  return (
    <Field
      label="Query"
      description="SQL returning one row per variable value. Return __text and __value columns to display one thing and store another."
    >
      <CodeEditor
        value={rawSql}
        language="sql"
        onChange={setRawSql}
        onBlur={onBlur}
        height="120px"
        showLineNumbers={true}
        showMiniMap={false}
        monacoOptions={{ wordWrap: 'on', scrollBeyondLastLine: false }}
      />
    </Field>
  );
}
