import { Tag, Button } from '@blueprintjs/core';
import { useEffect, useState } from 'react';

import './index.css';

import { GetVersion } from '../../wailsjs/go/cfg/Config';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { ProxyState } from '../types';

import { PortInput } from './PortInput';
import { UninstallCADialog } from './UninstallCADialog';

export interface SettingsManagerProps {
  proxyState: ProxyState;
}
export function SettingsManager({ proxyState }: SettingsManagerProps) {
  const [state, setState] = useState({
    version: '',
  });

  useEffect(() => {
    (async () => {
      const version = await GetVersion();
      setState({ ...state, version });
    })();
  }, []);

  return (
    <div className="settings-manager">
      <div className="settings-manager__section--advanced">
        <Tag large intent="warning" fill className="settings-manager__section-header">
          Advanced
        </Tag>

        <div className="settings-manager__section-body">
          <PortInput />
          <UninstallCADialog proxyState={proxyState} />
        </div>
      </div>

      <div className="settings-manager__about bp5-text-muted">
        <div>
          <strong>Zen</strong>
        </div>
        <div>Your Comprehensive Ad-Blocker and Privacy Guard</div>
        <div>Version: {state.version}</div>
        <div>© 2024 Ansar Smagulov</div>
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
