import { MenuItem } from '@blueprintjs/core';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { ImportCustomFilterLists } from '../../wailsjs/go/app/App';
import { AppToaster } from '../common/toaster';

export function ImportFilterList({ onAdd }: { onAdd: () => void }) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);

  const handleImport = async () => {
    setLoading(true);
    try {
      await ImportCustomFilterLists();
      AppToaster.show({
        message: t('importFilterList.successMessage'),
        intent: 'success',
      });
      onAdd();
    } catch (error) {
      AppToaster.show({
        message: t('importFilterList.errorMessage', { error }),
        intent: 'danger',
      });
    } finally {
      setLoading(false);
    }
  };

  return <MenuItem icon="download" text={t('importFilterList.import')} onClick={handleImport} disabled={loading} />;
}
