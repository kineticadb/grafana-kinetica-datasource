import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput, useStyles2 } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps, GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';
import { KineticaDataSourceOptions, KineticaSecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<KineticaDataSourceOptions, KineticaSecureJsonData> {}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    margin-bottom: ${theme.spacing(2)};
  `,
});

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields } = options;
  const secureJsonData = (options.secureJsonData || {}) as KineticaSecureJsonData;
  const styles = useStyles2(getStyles);

  const onDataChange = (key: keyof KineticaDataSourceOptions, value: string) => {
    onOptionsChange({ ...options, jsonData: { ...jsonData, [key]: value } });
  };

  const onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: { ...secureJsonData, password: event.target.value },
    });
  };

  const onResetPassword = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: { ...secureJsonFields, password: false },
      secureJsonData: { ...secureJsonData, password: '' },
    });
  };

  return (
    <div className={styles.container}>
      <InlineField label="URL" labelWidth={12}>
        <Input value={jsonData.url || ''}
        onChange={(e) => onDataChange('url', e.currentTarget.value)}
        placeholder="http://<host>:9191"
        width={40}/>
      </InlineField>
      <InlineField label="User" labelWidth={12}>
        <Input value={jsonData.username || ''} onChange={(e) => onDataChange('username', e.currentTarget.value)} width={20}/>
      </InlineField>
      <InlineField label="Password" labelWidth={12}>
        <SecretInput isConfigured={Boolean(secureJsonFields?.password)} value={secureJsonData.password || ''} onReset={onResetPassword} onChange={onPasswordChange} width={20}/>
      </InlineField>
    </div>
  );
}
