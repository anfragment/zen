import { Button, FormGroup, InputGroup } from '@blueprintjs/core';
import { useState, useRef } from 'react';

import { AddFilterList } from '../../wailsjs/go/cfg/Config';
import { AppToaster } from '../common/toaster';

import { FilterListType } from './types';

export function CreateFilterList({ onAdd }: { onAdd: () => void }) {
  const urlRef = useRef<HTMLInputElement>(null);
  const nameRef = useRef<HTMLInputElement>(null);
  const [loading, setLoading] = useState(false);

  return (
    <div className="filter-lists__create-filter-list">
      <FormGroup label="URL" labelFor="url" labelInfo="(required)">
        <InputGroup id="url" placeholder="https://example.com/filter-list.txt" required type="url" inputRef={urlRef} />
      </FormGroup>

      <FormGroup label="Name" labelFor="name" labelInfo="(optional)">
        <InputGroup id="name" placeholder="Example filter list" type="text" inputRef={nameRef} />
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
          onAdd();
        }}
        loading={loading}
      >
        Add filter list
      </Button>
    </div>
  );
}
