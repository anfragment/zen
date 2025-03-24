import { Radio, RadioGroup, FormGroup } from '@blueprintjs/core';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { GetUpdatePolicy, SetUpdatePolicy } from '../../../wailsjs/go/cfg/Config';
import { cfg } from '../../../wailsjs/go/models';

export function UpdatePolicyRadioGroup() {
  const { t } = useTranslation();
  const [state, setState] = useState({
    policy: '',
  });

  useEffect(() => {
    (async () => {
      const policy = await GetUpdatePolicy();
      setState((prev) => ({
        ...prev,
        policy,
      }));
    })();
  }, []);

  return (
    <FormGroup helperText={t('settings.updates.description')}>
      <RadioGroup
        label={t('settings.updates.choosePolicy')}
        onChange={async (e: any) => {
          const p = e.target.value;
          if (p) {
            await SetUpdatePolicy(p);
            setState((prev) => ({
              ...prev,
              policy: p,
            }));
          }
        }}
        selectedValue={state.policy}
      >
        <Radio label={t('settings.updates.automatic') as string} value={cfg.UpdatePolicyType.AUTOMATIC} />
        <Radio label={t('settings.updates.prompt') as string} value={cfg.UpdatePolicyType.PROMPT} />
        <Radio label={t('settings.updates.disabled') as string} value={cfg.UpdatePolicyType.DISABLED} />
      </RadioGroup>
    </FormGroup>
  );
}
