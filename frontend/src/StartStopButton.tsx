import { Button } from '@blueprintjs/core';
import { useEffect } from 'react';

import { StartProxy, StopProxy } from '../wailsjs/go/app/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

import { AppToaster } from './common/toaster';
import { ProxyState } from './types';

const PROXY_CHANNEL = 'proxy:action';

export interface StartStopButtonProps {
  proxyState: ProxyState;
  setProxyState: (state: ProxyState) => void;
}

enum ProxyActionKind {
  Starting = 'starting',
  On = 'on',
  Error = 'error',
}

interface ProxyAction {
  kind: ProxyActionKind;
  error?: string;
}

export function StartStopButton({ proxyState, setProxyState }: StartStopButtonProps) {
  useEffect(() => {
    const cancel = EventsOn(PROXY_CHANNEL, (action: ProxyAction) => {
      switch (action.kind) {
        case ProxyActionKind.Starting:
          setProxyState('loading');
          break;
        case ProxyActionKind.On:
          setProxyState('on');
          break;
        case ProxyActionKind.Error:
          AppToaster.show({
            message: `Failed to start proxy: ${action.error}`,
            intent: 'danger',
          });
          break;
        default:
          console.log('unknown proxy action', action);
      }
    });

    return cancel;
  }, []);

  const startProxy = async () => {
    setProxyState('loading');
    try {
      await StartProxy();
    } catch (err) {
      AppToaster.show({
        message: `Failed to start proxy: ${err}`,
        intent: 'danger',
      });
      setProxyState('off');
      return;
    }
    setProxyState('on');
  };
  const stopProxy = async () => {
    setProxyState('loading');
    try {
      await StopProxy();
    } catch (err) {
      AppToaster.show({
        message: `Failed to stop proxy: ${err}`,
        intent: 'danger',
      });
    } finally {
      setProxyState('off');
    }
  };

  return (
    <Button
      onClick={proxyState === 'off' ? startProxy : stopProxy}
      fill
      intent="primary"
      className="footer"
      large
      loading={proxyState === 'loading'}
    >
      {proxyState === 'off' ? 'Start' : 'Stop'}
    </Button>
  );
}
