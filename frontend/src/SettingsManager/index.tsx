import { Button, Tag } from '@blueprintjs/core';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import './index.css';

import { IsNoSelfUpdate } from '../../wailsjs/go/app/App';
import { GetVersion } from '../../wailsjs/go/cfg/Config';
import { BrowserOpenURL } from '../../wailsjs/runtime';
import { ProxyState } from '../types';

import { AutostartSwitch } from './AutostartSwitch';
import { ExportLogsButton } from './ExportLogsButton';
import { IgnoredHostsInput } from './IgnoredHostsInput';
import { LocaleSelector } from './LocaleSelector';
import { PortInput } from './PortInput';
import { UninstallCADialog } from './UninstallCADialog';
import { UpdatePolicyRadioGroup } from './UpdatePolicyRadioGroup';

const GITHUB_URL = 'https://github.com/anfragment/zen';
const CHANGELOG_URL = `${GITHUB_URL}/blob/master/CHANGELOG.md`;

export interface SettingsManagerProps {
  proxyState: ProxyState;
}
export function SettingsManager({ proxyState }: SettingsManagerProps) {
  const { t } = useTranslation();
  const [state, setState] = useState({
    version: '',
    updatePolicy: '',
    showUpdateRadio: false,
  });

  useEffect(() => {
    (async () => {
      const [version, noSelfUpdate] = await Promise.all([GetVersion(), IsNoSelfUpdate()]);

      setState((prev) => ({
        ...prev,
        showUpdateRadio: !noSelfUpdate,
        version,
      }));
    })();
  }, []);

  return (
    <div className="settings-manager">
      <div className="settings-manager__section--app">
        <Tag large intent="primary" fill className="settings-manager__section-header">
          {t('settings.sections.app')}
        </Tag>

        <div className="settings-manager__section-body">
          <LocaleSelector />
          <AutostartSwitch />
          {state.showUpdateRadio && <UpdatePolicyRadioGroup />}
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
          <strong>Zen</strong>
        </div>
        <div>{t('settings.about.tagline')}</div>
        <div>
          {t('settings.about.version')}: {state.version}
          <span className="settings-manager__about-changelog">
            ({/* eslint-disable-next-line jsx-a11y/anchor-is-valid  */}
            <a
              onClick={() => BrowserOpenURL(CHANGELOG_URL)}
              tabIndex={0}
              role="button"
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  BrowserOpenURL(CHANGELOG_URL);
                }
              }}
            >
              changelog
            </a>
            )
          </span>
        </div>
        <div>© 2025 Ansar Smagulov</div>
        <Button
          minimal
          small
          icon="git-branch"
          className="settings-manager__about-github-button"
          onClick={() => BrowserOpenURL(GITHUB_URL)}
        >
          {t('settings.about.github')}
        </Button>
      </div>
    </div>
  );
}
