import { MenuItem } from '@blueprintjs/core';
import { useState } from 'react';

import { ExportCustomFilterLists } from '../../wailsjs/go/app/App';
import { AppToaster } from '../common/toaster';

export function ExportFilterList() {
  const [loading, setLoading] = useState(false);

  const handleExport = async () => {
    setLoading(true);
    try {
      await ExportCustomFilterLists();
      AppToaster.show({
        message: 'Custom filter lists exported successfully',
        intent: 'success',
      });
    } catch (error) {
      AppToaster.show({
        message: `${error}`,
        intent: 'danger',
      });
    } finally {
      setLoading(false);
    }
  };

  return <MenuItem icon="upload" text="Export" onClick={handleExport} disabled={loading} />;
}
