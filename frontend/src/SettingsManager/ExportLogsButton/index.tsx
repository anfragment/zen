import { Button, Tooltip } from '@blueprintjs/core';

import { OpenLogsFolder } from '../../../wailsjs/go/app/App';

import './index.css';

export function ExportLogsButton() {
  return (
    <Tooltip content="Open logs directory">
      <Button onClick={OpenLogsFolder} intent="primary" className="export-logs__button">
        Export logs
      </Button>
    </Tooltip>
  );
}
