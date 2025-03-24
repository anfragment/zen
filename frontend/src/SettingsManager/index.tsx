import { Tag, Button } from '@blueprintjs/core';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import './index.css';

import { GetVersion } from '../../wailsjs/go/cfg/Config';
import { BrowserOpenURL } from '../../wailsjs/runtime';
import { ProxyState } from '../types';

import { AutostartSwitch } from './AutostartSwitch';
import { ExportLogsButton } from './ExportLogsButton';
import { IgnoredHostsInput } from './IgnoredHostsInput';
import { PortInput } from './PortInput';
import { UninstallCADialog } from './UninstallCADialog';
import { LanguageSelector } from './LanguageSelector';

export interface SettingsManagerProps {
  proxyState: ProxyState;
}
export function SettingsManager({ proxyState }: SettingsManagerProps) {
  const { t } = useTranslation();
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
      <div className="settings-manager__section--app">
        <Tag large intent="primary" fill className="settings-manager__section-header">
          {t('settings.sections.app')}
        </Tag>

        <div className="settings-manager__section-body">
          <LanguageSelector />
        </div>
        <div className="settings-manager__section-body">
          <AutostartSwitch />
        </div>

        <div className="settings-manager__section-body">
          <ExportLogsButton />
        </div>
      </div>

      <div className="settings-manager__section--advanced">
        <Tag large intent="warning" fill className="settings-manager__section-header">
          {t('settings.sections.advanced')}
        </Tag>

        <div className="settings-manager__section-body">
          <PortInput />
          <IgnoredHostsInput />
          <UninstallCADialog proxyState={proxyState} />
        </div>
      </div>

      <div className="settings-manager__about bp5-text-muted">
        <div>
          <strong>{t('app.title')}</strong>
        </div>
        <div>{t('settings.about.tagline')}</div>
        <div>
          {t('settings.about.version')}: {state.version}
        </div>
        <div>© 2025 Ansar Smagulov</div>
        <Button
          minimal
          small
          icon="git-branch"
          className="settings-manager__about-github-button"
          onClick={() => BrowserOpenURL('https://github.com/anfragment/zen')}
        >
          {t('settings.about.github')}
        </Button>
      </div>
    </div>
  );
}
