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
  Started = 'started',
  StartError = 'startError',
  Stopping = 'stopping',
  Stopped = 'stopped',
  StopError = 'stopError',
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
        case ProxyActionKind.Started:
          setProxyState('on');
          break;
        case ProxyActionKind.StartError:
          AppToaster.show({
            message: `Failed to start proxy: ${action.error}`,
            intent: 'danger',
          });
          setProxyState('on'); // Still worth it to give the option to shut down in case the error is recoverable.
          break;
        case ProxyActionKind.Stopping:
          setProxyState('loading');
          break;
        case ProxyActionKind.Stopped:
          setProxyState('off');
          break;
        case ProxyActionKind.StopError:
          AppToaster.show({
            message: `Failed to stop proxy: ${action.error}`,
            intent: 'danger',
          });
          setProxyState('off');
          break;
        default:
          console.log('unknown proxy action', action);
      }
    });

    return cancel;
  }, []);

  return (
    <Button
      onClick={proxyState === 'off' ? StartProxy : StopProxy}
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
