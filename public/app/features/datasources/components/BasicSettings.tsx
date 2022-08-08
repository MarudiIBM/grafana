import React, { useEffect, useState } from 'react';

import { selectors } from '@grafana/e2e-selectors';
import { getDataSourceSrv } from '@grafana/runtime';
import { InlineField, InlineSwitch, Input, Alert } from '@grafana/ui';

const useInitHasAlertingEnabled = (dataSourceName: string) => {
  const [hasAlertingEnabled, setHasAlertingEnabled] = useState(false);

  useEffect(() => {
    const initDataSourceAlertingEnabled = async () => {
      const ds = await getDataSourceSrv()?.get(dataSourceName);
      setHasAlertingEnabled(Boolean(ds?.meta?.alerting ?? false));
    };
    dataSourceName && initDataSourceAlertingEnabled();
  }, [dataSourceName]);
  return hasAlertingEnabled;
};

export interface Props {
  dataSourceName: string;
  isDefault: boolean;
  onNameChange: (name: string) => void;
  onDefaultChange: (value: boolean) => void;
}
export function AlertingEnabled({ enabled }: { enabled: boolean }) {
  return (
    <Alert
      severity="info"
      title={enabled ? 'This data source supports alerting' : 'This data source does not support alerting'}
    />
  );
}

export function BasicSettings({ dataSourceName, isDefault, onDefaultChange, onNameChange }: Props) {
  const hasAlertingEnabled = useInitHasAlertingEnabled(dataSourceName);

  return (
    <div className="gf-form-group" aria-label="Datasource settings page basic settings">
      <AlertingEnabled enabled={hasAlertingEnabled} />
      <div className="gf-form-inline">
        {/* Name */}
        <div className="gf-form max-width-30">
          <InlineField
            label="Name"
            tooltip="The name is used when you select the data source in panels. The default data source is
              'preselected in new panels."
            grow
          >
            <Input
              id="basic-settings-name"
              type="text"
              value={dataSourceName}
              placeholder="Name"
              onChange={(event) => onNameChange(event.currentTarget.value)}
              required
              aria-label={selectors.pages.DataSource.name}
            />
          </InlineField>
        </div>

        {/* Is Default */}
        <InlineField label="Default" labelWidth={8}>
          <InlineSwitch
            id="basic-settings-default"
            value={isDefault}
            onChange={(event: React.FormEvent<HTMLInputElement>) => {
              onDefaultChange(event.currentTarget.checked);
            }}
          />
        </InlineField>
      </div>
    </div>
  );
}
