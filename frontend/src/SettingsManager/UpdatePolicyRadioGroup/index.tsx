import { Radio, RadioGroup, FormGroup } from '@blueprintjs/core';
import { useEffect, useState } from 'react';

import { GetUpdatePolicy, SetUpdatePolicy } from '../../../wailsjs/go/cfg/Config';
import { cfg } from '../../../wailsjs/go/models';

export function UpdatePolicyRadioGroup() {
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
    <FormGroup
      helperText={
        <>
          If &quot;Automatic updates&quot; is selected, the app will automatically check for and apply updates on
          launch.
        </>
      }
    >
      <RadioGroup
        label="Choose how updates are installed"
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
        <Radio label="Automatic updates" value={cfg.UpdatePolicyType.AUTOMATIC} />
        <Radio label="Ask before updating" value={cfg.UpdatePolicyType.PROMPT} />
        <Radio label="Disable updates" value={cfg.UpdatePolicyType.DISABLED} />
      </RadioGroup>
    </FormGroup>
  );
}
