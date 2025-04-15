import { Button, Dialog, DialogBody, DialogFooter, Tooltip } from '@blueprintjs/core';
import { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import './index.css';

import { UninstallCA } from '../../../wailsjs/go/app/App';
import { AppToaster } from '../../common/toaster';
import { ProxyState } from '../../types';

export interface UninstallCADialogProps {
  proxyState: ProxyState;
}
export function UninstallCADialog({ proxyState }: UninstallCADialogProps) {
  const { t } = useTranslation();
  const [state, setState] = useState({
    isOpen: false,
    loading: false,
  });

  return (
    <>
      <Tooltip content={proxyState !== 'off' ? (t('settings.ca.stopProxyTooltip') as string) : undefined}>
        <Button
          disabled={proxyState !== 'off'}
          onClick={() => setState((state) => ({ ...state, isOpen: true }))}
          intent="danger"
          className="uninstall-ca-dialog__button"
        >
          {t('settings.ca.uninstallButton')}
        </Button>
      </Tooltip>

      <Dialog
        isOpen={state.isOpen}
        onClose={() => setState((state) => ({ ...state, isOpen: false }))}
        title={
          <Trans
            i18nKey="settings.ca.confirmTitle"
            components={{
              br: <br />,
            }}
          />
        }
        isCloseButtonShown={!state.loading}
        className="uninstall-ca-dialog"
      >
        <DialogBody>
          <p>
            {t('settings.ca.usefulIf')}:
            <ul>
              <li>{t('settings.ca.reasons.uninstall')}</li>
              <li>{t('settings.ca.reasons.malfunctioning')}</li>
            </ul>
            {t('settings.ca.reinstallInfo')}
          </p>
        </DialogBody>
        <DialogFooter
          actions={
            <Button
              intent="danger"
              loading={state.loading}
              onClick={async () => {
                setState((state) => ({ ...state, loading: true }));
                try {
                  await UninstallCA();
                  AppToaster.show({
                    message: t('settings.ca.successMessage'),
                    intent: 'success',
                  });
                } catch (err) {
                  AppToaster.show({
                    message: t('settings.ca.errorMessage', { error: err }),
                    intent: 'danger',
                  });
                } finally {
                  setState((state) => ({ ...state, isOpen: false, loading: false }));
                }
              }}
            >
              {t('settings.ca.uninstallConfirm')}
            </Button>
          }
        />
      </Dialog>
    </>
  );
}
