import { Button, ButtonGroup, Icon, IconSize, FocusStyleManager, NonIdealState } from '@blueprintjs/core';
import { useState, useEffect } from 'react';

import { StartProxy, StopProxy } from '../wailsjs/go/app/App';

import './App.css';
import { AppToaster } from './common/toaster';
import { FilterLists } from './FilterLists';
import { RequestLog } from './RequestLog';
import { SettingsManager } from './SettingsManager';
import { ProxyState } from './types';

function App() {
  useEffect(() => {
    FocusStyleManager.onlyShowFocusOnTabs();
  }, []);

  const [proxyState, setProxyState] = useState<ProxyState>('off');
  const [activeTab, setActiveTab] = useState<'home' | 'filterLists' | 'settings'>('home');

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
        {activeTab === 'settings' && <SettingsManager proxyState={proxyState} />}
      </div>

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
    </div>
  );
}

export default App;
