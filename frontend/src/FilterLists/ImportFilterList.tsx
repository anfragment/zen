import { Intent, MenuItem } from '@blueprintjs/core';
import { useState } from 'react';

import { ImportFilterList as ImportFilterListBackend } from '../../wailsjs/go/files/Files';
import { AppToaster } from '../common/toaster';

export function ImportFilterList({ onAdd }: { onAdd: () => void }) {
  const [loading, setLoading] = useState(false);

  const handleImport = async () => {
    setLoading(true);

    try {
      const result = await ImportFilterListBackend();

      const blob = new Blob([JSON.stringify(result)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'filter-lists.json';
      document.body.appendChild(link);
      link.click();

      document.body.removeChild(link);
      URL.revokeObjectURL(url);

      onAdd();
    } catch (error) {
      AppToaster.show({
        message: error instanceof Error ? error.message : 'Failed to export filter lists',
        intent: Intent.DANGER,
      });
    } finally {
      setLoading(false);
    }
  };

  return <MenuItem icon="download" text="Import" onClick={handleImport} disabled={loading} />;
}
