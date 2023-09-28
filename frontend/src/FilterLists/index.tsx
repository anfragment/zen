import {
  Tab, Tabs, Spinner, SpinnerSize, Switch,
} from '@blueprintjs/core';
import { useState, useEffect } from 'react';

import { GetFilterLists, ToggleFilterList } from '../../wailsjs/go/config/config';
import { config } from '../../wailsjs/go/models';

import './index.css';

// TODO: it would be nice to have a way to share types between the frontend and backend
// investigate whether this is possible
type FilterListType = 'general' | 'ads' | 'privacy';

export function FilterLists() {
  const [state, setState] = useState<{
    filterLists: config.filterList[];
    loading: boolean;
  }>({
    filterLists: [],
    loading: true,
  });

  const fetchLists = async () => {
    const filterLists = await GetFilterLists();
    setState({ ...state, filterLists, loading: false });
  };

  useEffect(() => {
    (async () => {
      await fetchLists();
    })();
  }, []);

  const [tab, setTab] = useState<FilterListType>('general');

  return (
    <>
      <Tabs
        id="FilterLists"
        onChange={(id) => setTab(id as FilterListType)}
        selectedTabId={tab}
        fill
      >
        <Tab id="general" title="General" />
        <Tab id="ads" title="Ads" />
        <Tab id="privacy" title="Privacy" />
      </Tabs>

      {state.loading && <Spinner size={SpinnerSize.SMALL} className="filter-lists__spinner" />}

      {state.filterLists.filter((filterList) => filterList.type === tab).map((filterList) => (
        <ListItem
          key={filterList.url}
          filterList={filterList}
          onToggle={fetchLists}
        />
      ))}
    </>
  );
}

function ListItem({ filterList, onToggle }: { filterList: config.filterList, onToggle: () => void }) {
  const [loading, setLoading] = useState(false);
  return (
    <div className="filter-lists__list">
      <div className="filter-lists__list-header">
        <h3 className="filter-lists__list-name">{filterList.name}</h3>
        <Switch
          checked={filterList.enabled}
          disabled={loading}
          onChange={async (e) => {
            setLoading(true);
            await ToggleFilterList(filterList.url, e.currentTarget.checked);
            setLoading(false);
            onToggle();
          }}
          large
          className="filter-lists__list-switch"
        />
      </div>
      <p className="bp5-text-muted filter-lists__list-url">
        {filterList.url}
      </p>
    </div>
  );
}
