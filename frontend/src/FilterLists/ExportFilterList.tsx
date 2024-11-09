import { MenuItem } from '@blueprintjs/core';
import { useState } from 'react';

import { ExportCustomFilterLists } from '../../wailsjs/go/app/App';
import { AppToaster } from '../common/toaster';

export function ExportFilterList() {
  const [loading, setLoading] = useState(false);

  const handleExport = async () => {
    setLoading(true);
    const error = await ExportCustomFilterLists();
    if (error) {
      AppToaster.show({
        message: `Failed to export custom filter lists: ${error}`,
        intent: 'danger',
      });
    } else {
      AppToaster.show({
        message: 'Custom filter lists exported successfully',
        intent: 'success',
      });
    }
    setLoading(false);
  };

  return <MenuItem icon="upload" text="Export" onClick={handleExport} disabled={loading} />;
}
