import { Button, ButtonGroup, Icon, IconSize, FocusStyleManager, NonIdealState } from '@blueprintjs/core';
import { useState, useEffect } from 'react';

import './App.css';

import { FilterLists } from './FilterLists';
import { MyRules } from './MyRules';
import { RequestLog } from './RequestLog';
import { SettingsManager } from './SettingsManager';
import { StartStopButton } from './StartStopButton';
import { ProxyState } from './types';

function App() {
  useEffect(() => {
    FocusStyleManager.onlyShowFocusOnTabs();
  }, []);

  const [proxyState, setProxyState] = useState<ProxyState>('off');
  const [activeTab, setActiveTab] = useState<'home' | 'filterLists' | 'myRules' | 'settings'>('home');

  return (
    <div id="App">
      <div className="heading">
        <h1 className="heading__logo">
          <Icon icon="shield" size={IconSize.LARGE} />
          ZEN
        </h1>
      </div>
      <ButtonGroup fill minimal className="tabs">
        <Button icon="circle" active={activeTab === 'home'} onClick={() => setActiveTab('home')}>
          Home
        </Button>
        <Button icon="filter" active={activeTab === 'filterLists'} onClick={() => setActiveTab('filterLists')}>
          Filter lists
        </Button>
        <Button icon="code" active={activeTab === 'myRules'} onClick={() => setActiveTab('myRules')}>
          My rules
        </Button>
        <Button icon="settings" active={activeTab === 'settings'} onClick={() => setActiveTab('settings')}>
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
        {activeTab === 'myRules' && <MyRules />}
        {activeTab === 'settings' && <SettingsManager proxyState={proxyState} />}
      </div>

      <StartStopButton proxyState={proxyState} setProxyState={setProxyState} />
    </div>
  );
}

export default App;
