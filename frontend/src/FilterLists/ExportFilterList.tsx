import { Intent, MenuItem } from '@blueprintjs/core';
import { useState } from 'react';

import { ExportFilterList as ExportFilterListBackend } from '../../wailsjs/go/files/FileExport';
import { AppToaster } from '../common/toaster';


export function ExportFilterList() {
  const [loading, setLoading] = useState(false);

  const handleExport = async() => {
    setLoading(true);
    try {
      const result = await ExportFilterListBackend();

      const blob = new Blob([JSON.stringify(result)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href =  url;
      link.download = 'filter-lists.json';
      document.body.appendChild(link);
      link.click();

      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (error) {
      AppToaster.show({
        message: error instanceof Error ? error.message : 'Failed to export filter lists',
        intent: Intent.DANGER,
      });
    } finally {
      setLoading(false);
    }
  };

  return <MenuItem 
    icon="download"
    text="Export Custom Filter Lists"
    onClick={handleExport}
    disabled={loading}
  />;
}
