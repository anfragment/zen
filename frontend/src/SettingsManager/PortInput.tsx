import { FormGroup, NumericInput, Tooltip, Position } from '@blueprintjs/core';
import { useEffect, useState } from 'react';
import { useDebouncedCallback } from 'use-debounce';

import { GetPort, SetPort } from '../../wailsjs/go/config/config';

import './PortInput.css';

export function PortInput() {
  const [state, setState] = useState({
    port: 0,
    loading: true,
  });

  useEffect(() => {
    (async () => {
      const port = await GetPort();
      setState({ ...state, port, loading: false });
    })();
  }, []);

  const setPort = useDebouncedCallback(async (port: number) => {
    await SetPort(port);
  }, 500);

  return (
    <FormGroup label="Port" labelFor="port" helperText="Zen will listen on this port for requests">
      <Tooltip content="The port the proxy server will listen on (0 for random). Using a port below 1024 may require elevated privileges.">
        <NumericInput
          buttonPosition={Position.RIGHT}
          id="port"
          min={0}
          max={65535}
          value={state.port}
          onValueChange={(port) => {
            setState({ ...state, port });
            setPort(port);
          }}
          disabled={state.loading}
          className="port-input"
        />
      </Tooltip>
    </FormGroup>
  );
}
