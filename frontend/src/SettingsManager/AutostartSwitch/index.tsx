import { Switch, FormGroup } from '@blueprintjs/core';
import { useCallback, useEffect, useState } from 'react';

import { IsEnabled, Enable, Disable } from '../../../wailsjs/go/autostart/Manager';
import { AppToaster } from '../../common/toaster';

export function AutostartSwitch() {
  const [state, setState] = useState({
    enabled: false,
    loading: true,
  });

  useEffect(() => {
    (async () => {
      const enabled = await IsEnabled();
      setState({ ...state, enabled, loading: false });
    })();
  }, []);

  const disable = useCallback(() => {
    (async () => {
      setState((state) => ({ ...state, loading: true }));
      try {
        await Disable();
      } catch (err) {
        AppToaster.show({
          message: `Failed to disable autostart: ${err}`,
          intent: 'danger',
        });
        setState((state) => ({ ...state, loading: false }));
        return;
      }
      setState((state) => ({ ...state, enabled: false, loading: false }));
    })();
  }, []);
  const enable = useCallback(() => {
    (async () => {
      setState((state) => ({ ...state, loading: true }));
      try {
        await Enable();
      } catch (err) {
        AppToaster.show({
          message: `Failed to enable autostart: ${err}`,
          intent: 'danger',
        });
        setState((state) => ({ ...state, loading: false }));
        return;
      }
      setState((state) => ({ ...state, enabled: true, loading: false }));
    })();
  }, []);

  return (
    <FormGroup label="Autostart" labelFor="autostart" helperText={<>Start Zen automatically when you log in.</>}>
      <Switch
        id="autostart"
        checked={state.enabled}
        large
        disabled={state.loading}
        onClick={() => {
          if (state.enabled) {
            disable();
          } else {
            enable();
          }
        }}
      />
    </FormGroup>
  );
}
