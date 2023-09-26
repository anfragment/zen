import {
  Button, ButtonGroup, Icon, IconSize, FocusStyleManager,
} from '@blueprintjs/core';
import { useState, useEffect } from 'react';

import { StartProxy, StopProxy } from '../wailsjs/go/main/App';

import { FilterLists } from './FilterLists';

import './App.css';

function App() {
  useEffect(() => {
    FocusStyleManager.onlyShowFocusOnTabs();
  }, []);

  const [proxyState, setProxyState] = useState<{ state: 'on' | 'off'; loading: boolean; }>({
    state: 'off',
    loading: false,
  });
  const [activeTab, setActiveTab] = useState<'home' | 'filterLists' | 'settings'>('home');

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
      <div className="heading">
        <h1 className="heading__logo">
          <Icon
            icon="shield" size={IconSize.LARGE}
          />
          ZEN
        </h1>

        <div className="heading__status--active">
          <Icon icon="dot" />
          Active
        </div>
      </div>
      <ButtonGroup fill minimal>
        <Button
          icon="circle" active={activeTab === 'home'}
          onClick={() => setActiveTab('home')}
        >
          Home
        </Button>
        <Button
          icon="filter" active={activeTab === 'filterLists'}
          onClick={() => setActiveTab('filterLists')}
        >
          Filter lists
        </Button>
        <Button
          icon="settings" active={activeTab === 'settings'}
          onClick={() => setActiveTab('settings')}
        >
          Settings
        </Button>
      </ButtonGroup>

      {activeTab === 'home' && (
        <>
          home
        </>
      )}
      {activeTab === 'filterLists' && (
        <FilterLists />
      )}
      {activeTab === 'settings' && (
        <>
          settings
        </>
      )}

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
