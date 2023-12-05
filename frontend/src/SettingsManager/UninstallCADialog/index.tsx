import { Button, Dialog, DialogBody, DialogFooter, Tooltip } from '@blueprintjs/core';
import { useState } from 'react';

import './index.css';

import { UninstallCA } from '../../../wailsjs/go/certmanager/CertManager';
import { AppToaster } from '../../common/toaster';
import { ProxyState } from '../../types';

export interface UninstallCADialogProps {
  proxyState: ProxyState;
}
export function UninstallCADialog({ proxyState }: UninstallCADialogProps) {
  const [state, setState] = useState({
    isOpen: false,
    loading: false,
  });

  return (
    <>
      <Tooltip content={proxyState !== 'off' ? 'Stop the proxy to uninstall the CA' : undefined}>
        <Button
          disabled={proxyState !== 'off'}
          onClick={() => setState((state) => ({ ...state, isOpen: true }))}
          intent="danger"
          className="uninstall-ca-dialog__button"
        >
          Uninstall CA
        </Button>
      </Tooltip>

      <Dialog
        isOpen={state.isOpen}
        onClose={() => setState((state) => ({ ...state, isOpen: false }))}
        title="Are you sure you want to uninstall the CA?"
        isCloseButtonShown={!state.loading}
        className="uninstall-ca-dialog"
      >
        <DialogBody>
          <p>
            This can be useful if:
            <ul>
              <li>You want to uninstall Zen completely.</li>
              <li>
                You suspect that your CA installation is malfunctioning. This issue may manifest as a browser error when
                visiting HTTPS websites, or as other applications being unable to connect to the internet.
              </li>
            </ul>
            If you start Zen again, a new CA will be installed.
          </p>
        </DialogBody>
        <DialogFooter
          actions={
            <Button
              intent="danger"
              loading={state.loading}
              onClick={async () => {
                setState((state) => ({ ...state, loading: true }));
                const err = await UninstallCA();
                if (err) {
                  AppToaster.show({
                    message: `Failed to uninstall CA: ${err}`,
                    intent: 'danger',
                  });
                } else {
                  AppToaster.show({
                    message: 'CA uninstalled successfully',
                    intent: 'success',
                  });
                }
                setState((state) => ({ ...state, isOpen: false, loading: false }));
              }}
            >
              Uninstall
            </Button>
          }
        />
      </Dialog>
    </>
  );
}
