import {
  Button,
  ButtonGroup,
  Icon,
  IconSize,
  FocusStyleManager,
  NonIdealState,
} from '@blueprintjs/core';
import { useState, useEffect } from 'react';

import { StartProxy, StopProxy } from '../wailsjs/go/main/App';

import './App.css';
import { FilterLists } from './FilterLists';
import { RequestLog } from './RequestLog';
import { SettingsManager } from './SettingsManager';

function App() {
  useEffect(() => {
    FocusStyleManager.onlyShowFocusOnTabs();
  }, []);

  const [proxyState, setProxyState] = useState<'on' | 'off' | 'loading'>('off');
  const [activeTab, setActiveTab] = useState<
    'home' | 'filterLists' | 'settings'
  >('home');

  const start = async () => {
    setProxyState('loading');
    await StartProxy();
    setProxyState('on');
  };
  const stop = async () => {
    setProxyState('loading');
    await StopProxy();
    setProxyState('off');
  };

  return (
    <div id="App" className="bp5-dark">
      <div className="heading">
        <h1 className="heading__logo">
          <Icon icon="shield" size={IconSize.LARGE} />
          ZEN
        </h1>
      </div>
      <ButtonGroup fill minimal className="tabs">
        <Button
          icon="circle"
          active={activeTab === 'home'}
          onClick={() => setActiveTab('home')}
        >
          Home
        </Button>
        <Button
          icon="filter"
          active={activeTab === 'filterLists'}
          onClick={() => setActiveTab('filterLists')}
        >
          Filter lists
        </Button>
        <Button
          icon="settings"
          active={activeTab === 'settings'}
          onClick={() => setActiveTab('settings')}
        >
          Settings
        </Button>
      </ButtonGroup>

      <div className="content">
        <div style={{ display: activeTab === 'home' ? 'block' : 'none' }}>
          {proxyState === 'off' ? (
            <NonIdealState
              icon="lightning"
              title="Activate the proxy to see blocked requests"
              description="The proxy is not active. Click the button below to activate it."
              className="request-log__non-ideal-state"
            />
          ) : (
            <RequestLog />
          )}
        </div>
        {activeTab === 'filterLists' && <FilterLists />}
        {activeTab === 'settings' && <SettingsManager />}
      </div>

      <Button
        onClick={proxyState === 'off' ? start : stop}
        fill
        intent="primary"
        className="footer"
        large
        loading={proxyState === 'loading'}
      >
        {proxyState === 'off' ? 'Start' : 'Stop'}
      </Button>
    </div>
  );
}

export default App;
