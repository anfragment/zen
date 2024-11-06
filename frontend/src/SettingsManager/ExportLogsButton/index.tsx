import { Button, Tooltip } from '@blueprintjs/core';
import { useState } from 'react';

import { OpenLogsDirectory } from '../../../wailsjs/go/app/App';
import './index.css';
import { AppToaster } from '../../common/toaster';

export function ExportLogsButton() {
  const [loading, setLoading] = useState(false);

  return (
    <Tooltip content="Open logs directory">
      <Button
        loading={loading}
        onClick={async () => {
          setLoading(true);
          try {
            await OpenLogsDirectory();
          } catch (err) {
            AppToaster.show({
              message: `Failed to open logs directory: ${err}`,
              intent: 'danger',
            });
          } finally {
            setLoading(false);
          }
        }}
        className="export-logs__button"
      >
        Export logs
      </Button>
    </Tooltip>
  );
}
