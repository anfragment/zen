import { NumericInput, FormGroup, Tag, Button } from '@blueprintjs/core';
import { useEffect, useState } from 'react';

import './index.css';

import { GetPort, SetPort, GetVersion } from '../../wailsjs/go/config/config';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { AppToaster } from '../common/toaster';
import { ProxyState } from '../types';

import { UninstallCADialog } from './UninstallCADialog';

export interface SettingsManagerProps {
  proxyState: ProxyState;
}
export function SettingsManager({ proxyState }: SettingsManagerProps) {
  const [state, setState] = useState({
    proxy: {
      port: 0,
    },
    version: '',
  });

  useEffect(() => {
    (async () => {
      const port = await GetPort();
      const version = await GetVersion();
      setState({ ...state, proxy: { port }, version });
    })();
  }, []);

  return (
    <div className="settings-manager">
      <div className="settings-manager__section--advanced">
        <Tag large intent="warning" fill className="settings-manager__section-header">
          Advanced
        </Tag>

        <div className="settings-manager__section-body">
          <FormGroup
            label="Port"
            labelFor="port"
            helperText={`
              The port the proxy server will listen on (0 for random).
              Be careful when using a port below 1024 as it may require elevated privileges.
            `}
          >
            <NumericInput
              id="port"
              min={0}
              max={65535}
              value={state.proxy.port}
              onValueChange={async (port) => {
                setState({ ...state, proxy: { port } });
                const err = await SetPort(port);
                if (err) {
                  AppToaster.show({
                    message: `Failed to set port: ${err}`,
                    intent: 'danger',
                  });
                }
              }}
            />
          </FormGroup>

          <UninstallCADialog proxyState={proxyState} />
        </div>
      </div>

      <div className="settings-manager__about bp5-text-muted">
        <div>
          <strong>Zen</strong>
        </div>
        <div>Your Comprehensive Ad-Blocker and Privacy Guard</div>
        <div>Version: {state.version}</div>
        <div>© 2023 Ansar Smagulov</div>
        <Button
          minimal
          small
          icon="git-branch"
          className="settings-manager__about-github-button"
          onClick={() => BrowserOpenURL('https://github.com/anfragment/zen')}
        >
          GitHub
        </Button>
      </div>
    </div>
  );
}
