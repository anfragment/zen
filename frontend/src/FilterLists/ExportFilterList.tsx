import { MenuItem } from '@blueprintjs/core';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { ExportCustomFilterLists } from '../../wailsjs/go/app/App';
import { AppToaster } from '../common/toaster';

export function ExportFilterList() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);

  const handleExport = async () => {
    setLoading(true);
    try {
      await ExportCustomFilterLists();
      AppToaster.show({
        message: t('exportFilterList.successMessage'),
        intent: 'success',
      });
    } catch (error) {
      AppToaster.show({
        message: t('exportFilterList.errorMessage', { error }),
        intent: 'danger',
      });
    } finally {
      setLoading(false);
    }
  };

  return <MenuItem icon="upload" text={t('exportFilterList.export')} onClick={handleExport} disabled={loading} />;
}
