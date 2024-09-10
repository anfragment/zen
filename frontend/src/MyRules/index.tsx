import { Button, TextArea } from '@blueprintjs/core';
import { useEffect, useState } from 'react';
import { useDebouncedCallback } from 'use-debounce';

import './index.css';
import { GetMyRules, SetMyRules } from '../../wailsjs/go/cfg/Config';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

const HELP_URL = 'https://github.com/anfragment/zen/blob/master/docs/external/how-to-rules.md';

export function MyRules() {
  const [state, setState] = useState({
    rules: '',
    loading: true,
  });

  useEffect(() => {
    (async () => {
      const filters = await GetMyRules();
      setState({ rules: filters.join('\n'), loading: false });
    })();
  }, []);

  const setFilters = useDebouncedCallback(async (rules: string) => {
    await SetMyRules(
      rules
        .split('\n')
        .map((f) => f.trim())
        .filter((f) => f.length > 0),
    );
  }, 500);

  return (
    <div className="my-rules">
      <div>
        <Button outlined icon="help" className="my-rules__help-button" onClick={() => BrowserOpenURL(HELP_URL)}>
          Help
        </Button>
      </div>
      <TextArea
        fill
        placeholder="Add your custom rules here..."
        className="my-rules__textarea"
        value={state.rules}
        onChange={(e) => {
          const { value } = e.target;
          setState({ ...state, rules: value });
          setFilters(value);
        }}
      />
    </div>
  );
}
