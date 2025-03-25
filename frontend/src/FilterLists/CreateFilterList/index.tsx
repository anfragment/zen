import { Button, Classes, FormGroup, InputGroup, Switch, Tooltip } from '@blueprintjs/core';
import { InfoSign } from '@blueprintjs/icons';
import { useState, useRef } from 'react';

import './index.css';

import { AddFilterList } from '../../../wailsjs/go/cfg/Config';
import { AppToaster } from '../../common/toaster';
import { FilterListType } from '../types';
import { useTranslation } from 'react-i18next';

export function CreateFilterList({ onAdd }: { onAdd: () => void }) {
  const { t } = useTranslation();
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
                {t('createFilterList.trustedCapabilities')} <code>trusted-</code>{' '}
                {t('createFilterList.trustedScriptlets')} <strong>{t('createFilterList.privacyWarning')}</strong>{' '}
                {t('createFilterList.ifHijacked')} {t('createFilterList.onlyEnable')}
              </span>
            }
            placement="top"
            minimal
            matchTargetWidth
          >
            <span className="create-filter-list__trusted-label">
              <span>{t('filterLists.trusted')}</span>
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
              message: t('createFilterList.addError', { error: err }),
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
        {t('createFilterList.addList')}
      </Button>
    </div>
  );
}
