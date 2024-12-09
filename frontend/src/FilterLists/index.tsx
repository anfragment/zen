import { Spinner, SpinnerSize, Switch, Button, MenuItem, Popover, Menu, Tag } from '@blueprintjs/core';
import { Select } from '@blueprintjs/select';
import { useState, useEffect } from 'react';

import { GetFilterLists, RemoveFilterList, ToggleFilterList } from '../../wailsjs/go/cfg/Config';
// eslint-disable-next-line import/order
import { type cfg } from '../../wailsjs/go/models';

import './index.css';

import { AppToaster } from '../common/toaster';

import { CreateFilterList } from './CreateFilterList';
import { ExportFilterList } from './ExportFilterList';
import { ImportFilterList } from './ImportFilterList';
import { FilterListType } from './types';

export function FilterLists() {
  const [state, setState] = useState<{
    filterLists: cfg.FilterList[];
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
    (() => {
      fetchLists();
    })();
  }, []);

  const [type, setType] = useState<FilterListType>(FilterListType.GENERAL);

  return (
    <>
      <div className="filter-lists__header">
        <Select
          items={Object.values(FilterListType)}
          itemRenderer={(item) => (
            <MenuItem
              key={item}
              text={
                <>
                  {item[0].toUpperCase() + item.slice(1)}
                  <span className="bp5-text-muted filter-lists__select-count">
                    ({state.filterLists.filter((filterList) => filterList.type === item && filterList.enabled).length}/
                    {state.filterLists.filter((filterList) => filterList.type === item).length})
                  </span>
                </>
              }
              onClick={() => {
                setType(item);
              }}
              active={item === type}
            />
          )}
          onItemSelect={(item) => {
            setType(item);
          }}
          popoverProps={{ minimal: true }}
          filterable={false}
        >
          <Button text={type[0].toUpperCase() + type.slice(1)} rightIcon="caret-down" />
        </Select>

        {type === FilterListType.CUSTOM && (
          <Popover
            content={
              <Menu>
                <ExportFilterList />
                <ImportFilterList onAdd={fetchLists} />
              </Menu>
            }
          >
            <Button icon="more" text="More" />
          </Popover>
        )}
      </div>

      {state.loading && <Spinner size={SpinnerSize.SMALL} className="filter-lists__spinner" />}

      {state.filterLists
        .filter((filterList) => filterList.type === type)
        .map((filterList) => (
          <ListItem
            key={filterList.url}
            filterList={filterList}
            showDelete={type === FilterListType.CUSTOM}
            onChange={fetchLists}
          />
        ))}

      {type === FilterListType.CUSTOM && <CreateFilterList onAdd={fetchLists} />}
    </>
  );
}

function ListItem({
  filterList,
  showDelete,
  onChange,
}: {
  filterList: cfg.FilterList;
  showDelete?: boolean;
  onChange?: () => void;
}) {
  const [switchLoading, setSwitchLoading] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);
  return (
    <div className="filter-lists__list">
      <div className="filter-lists__list-header">
        <h3 className="filter-lists__list-name">{filterList.name}</h3>
        <Switch
          checked={filterList.enabled}
          disabled={switchLoading}
          onChange={async (e) => {
            setSwitchLoading(true);
            const err = await ToggleFilterList(filterList.url, e.currentTarget.checked);
            if (err) {
              AppToaster.show({
                message: `Failed to toggle filter list: ${err}`,
                intent: 'danger',
              });
            }
            setSwitchLoading(false);
            onChange?.();
          }}
          large
          className="filter-lists__list-switch"
        />
      </div>
      {filterList.trusted ? (
        <Tag intent="success" className="filter-lists__list-trusted">
          Trusted
        </Tag>
      ) : null}

      <div className="bp5-text-muted filter-lists__list-url">{filterList.url}</div>

      {showDelete && (
        <Button
          icon="trash"
          intent="danger"
          small
          className="filter-lists__list-delete"
          loading={deleteLoading}
          onClick={async () => {
            setDeleteLoading(true);
            const err = await RemoveFilterList(filterList.url);
            if (err) {
              AppToaster.show({
                message: `Failed to remove filter list: ${err}`,
                intent: 'danger',
              });
            }
            setDeleteLoading(false);
            onChange?.();
          }}
        >
          Delete
        </Button>
      )}
    </div>
  );
}
