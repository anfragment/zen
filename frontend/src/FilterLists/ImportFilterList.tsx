import { MenuItem } from '@blueprintjs/core';
import { useState } from 'react';

import { ImportCustomFilterLists } from '../../wailsjs/go/app/App';
import { AppToaster } from '../common/toaster';

export function ImportFilterList({ onAdd }: { onAdd: () => void }) {
  const [loading, setLoading] = useState(false);

  const handleImport = async () => {
    setLoading(true);
    try {
      await ImportCustomFilterLists();
      AppToaster.show({
        message: 'Custom filter lists imported successfully',
        intent: 'success',
      });
      onAdd();
    } catch (error) {
      AppToaster.show({
        message: `${error}`,
        intent: 'danger',
      });
    } finally {
      setLoading(false);
    }
  };

  return <MenuItem icon="download" text="Import" onClick={handleImport} disabled={loading} />;
}
