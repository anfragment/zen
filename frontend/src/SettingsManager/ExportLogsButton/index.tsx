import { Button, Tooltip } from '@blueprintjs/core';

import { OpenLogsDirectory } from '../../../wailsjs/go/app/App';

import './index.css';

export function ExportLogsButton() {
  return (
    <Tooltip content="Open logs directory">
      <Button onClick={OpenLogsDirectory} className="export-logs__button">
        Export logs
      </Button>
    </Tooltip>
  );
}
