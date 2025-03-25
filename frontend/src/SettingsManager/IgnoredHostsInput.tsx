import { FormGroup, TextArea } from '@blueprintjs/core';
import { useEffect, useState } from 'react';
import { useDebouncedCallback } from 'use-debounce';

import { GetIgnoredHosts, SetIgnoredHosts } from '../../wailsjs/go/cfg/Config';

export function IgnoredHostsInput() {
  const [state, setState] = useState({
    ignoredHosts: '',
    loading: true,
  });

  useEffect(() => {
    (async () => {
      const ignoredHosts = await GetIgnoredHosts();
      setState({ ignoredHosts: (ignoredHosts ?? []).join('\n'), loading: false });
    })();
  }, []);

  const setIgnoredHosts = useDebouncedCallback(async (ignoredHosts: string) => {
    await SetIgnoredHosts(
      ignoredHosts
        .split('\n')
        .map((host) => host.trim())
        .filter((host) => host.length > 0),
    );
  }, 500);

  return (
    <FormGroup
      label="Ignored Hosts"
      labelFor="ignoredHosts"
      helperText={
        <>
          Hosts to exclude from proxying. Use this for services with certificate pinning or those disrupted by proxying.
          <br />
          Enter one host per line.
        </>
      }
    >
      <TextArea
        id="ignoredHosts"
        placeholder="example.com"
        className="settings-manager__ignored-hosts-input"
        value={state.ignoredHosts}
        onChange={(e) => {
          const { value } = e.target;
          setState({ ...state, ignoredHosts: value });
          setIgnoredHosts(value);
        }}
        disabled={state.loading}
        autoResize
        fill
      />
    </FormGroup>
  );
}
