import { MenuItem } from '@blueprintjs/core';
import { useState } from 'react';

import { ImportCustomFilterLists } from '../../wailsjs/go/app/App';
import { AppToaster } from '../common/toaster';

export function ImportFilterList({ onAdd }: { onAdd: () => void }) {
  const [loading, setLoading] = useState(false);

  const handleImport = async () => {
    setLoading(true);
    const error = await ImportCustomFilterLists();
    if (error) {
      AppToaster.show({
        message: `Failed to import custom filter lists: ${error}`,
        intent: 'danger',
      });
    } else {
      AppToaster.show({
        message: 'Custom filter lists imported successfully',
        intent: 'success',
      });
      onAdd();
    }
    setLoading(false);
  };

  return <MenuItem icon="download" text="Import" onClick={handleImport} disabled={loading} />;
}
