import { Button, Classes, FormGroup, InputGroup, Switch, Tooltip } from '@blueprintjs/core';
import { InfoSign } from '@blueprintjs/icons';
import { useState, useRef } from 'react';

import './index.css';

import { AddFilterList } from '../../../wailsjs/go/cfg/Config';
import { AppToaster } from '../../common/toaster';
import { FilterListType } from '../types';

export function CreateFilterList({ onAdd }: { onAdd: () => void }) {
  const urlRef = useRef<HTMLInputElement>(null);
  const nameRef = useRef<HTMLInputElement>(null);
  const [trusted, setTrusted] = useState(false);
  const [loading, setLoading] = useState(false);

  return (
    <div className="filter-lists__create-filter-list">
      <FormGroup label="URL" labelFor="url" labelInfo="(required)">
        <InputGroup id="url" placeholder="https://example.com/filter-list.txt" required type="url" inputRef={urlRef} />
      </FormGroup>

      <FormGroup label="Name" labelFor="name" labelInfo="(optional)">
        <InputGroup id="name" placeholder="Example filter list" type="text" inputRef={nameRef} />
      </FormGroup>

      <FormGroup
        label={
          <Tooltip
            content={
              <span className={Classes.TEXT_SMALL}>
                Trusted lists can use powerful blocking capabilities, such as <code>trusted-</code> scriptlets and
                JavaScript rules but <strong>may disrupt privacy and security</strong> if hijacked by a malicious party.
                Only enable this option for lists from sources you trust.
              </span>
            }
            placement="top"
            minimal
            matchTargetWidth
          >
            <span className="create-filter-list__trusted-label">
              <span>Trusted</span>
              <InfoSign className={Classes.TEXT_MUTED} size={12} />
            </span>
          </Tooltip>
        }
        labelFor="trusted"
      >
        <Switch
          id="trusted"
          large
          checked={trusted}
          onClick={(e) => {
            setTrusted(e.currentTarget.checked);
          }}
        />
      </FormGroup>

      <Button
        icon="add"
        intent="primary"
        fill
        onClick={async () => {
          if (!urlRef.current?.checkValidity()) {
            urlRef.current?.focus();
            return;
          }
          const url = urlRef.current?.value;
          const name = nameRef.current?.value || url;

          setLoading(true);
          const err = await AddFilterList({
            url,
            name,
            type: FilterListType.CUSTOM,
            enabled: true,
            trusted,
          });
          if (err) {
            AppToaster.show({
              message: `Failed to add filter list: ${err}`,
              intent: 'danger',
            });
          }
          setLoading(false);
          urlRef.current!.value = '';
          nameRef.current!.value = '';
          setTrusted(false);
          onAdd();
        }}
        loading={loading}
      >
        Add filter list
      </Button>
    </div>
  );
}
