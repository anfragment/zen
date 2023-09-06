import { Button } from '@blueprintjs/core';
import { useState } from 'react';

import { StartProxy, StopProxy } from '../wailsjs/go/main/App';

import './App.css';

function App() {
  const [proxyState, setProxyState] = useState<{ state: 'on' | 'off'; loading: boolean; }>({
    state: 'off',
    loading: false,
  });

  const start = async () => {
    setProxyState({ ...proxyState, loading: true });
    await StartProxy();
    setProxyState({ ...proxyState, state: 'on', loading: false });
  };
  const stop = async () => {
    setProxyState({ ...proxyState, loading: true });
    await StopProxy();
    setProxyState({ ...proxyState, state: 'off', loading: false });
  };

  return (
    <div id="App" className="bp5-dark">
      <h1 className="heading">ZEN</h1>

      <Button
        onClick={proxyState.state === 'off' ? start : stop}
        fill
        intent="primary"
        className="footer"
        large
        loading={proxyState.loading}
      >
        {proxyState.state === 'off' ? 'Start' : 'Stop'}
      </Button>
    </div>
  );
}

export default App;
