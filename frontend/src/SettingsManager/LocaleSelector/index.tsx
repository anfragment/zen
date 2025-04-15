import { Button, FormGroup, MenuItem } from '@blueprintjs/core';
import { ItemRenderer, Select } from '@blueprintjs/select';
import { useTranslation } from 'react-i18next';

import { changeLocale, getCurrentLocale, SupportedLocale } from '../../i18n';

interface LocaleItem {
  value: SupportedLocale;
  label: string;
}

const items: LocaleItem[] = [
  { value: 'en-US', label: 'English' },
  { value: 'de-DE', label: 'Deutsch' },
  { value: 'kk-KZ', label: 'Қазақша' },
  { value: 'ru-RU', label: 'Русский' },
];

export function LocaleSelector() {
  const { t } = useTranslation();

  const handleLocaleChange = async (item: LocaleItem) => {
    changeLocale(item.value);
  };

  const renderItem: ItemRenderer<LocaleItem> = (item, { handleClick, handleFocus, modifiers }) => {
    return (
      <MenuItem
        active={modifiers.active}
        key={item.value}
        onClick={handleClick}
        onFocus={handleFocus}
        roleStructure="listoption"
        text={item.label}
      />
    );
  };

  const currentLocale = items.find((item) => item.value === getCurrentLocale()) || items[0];

  return (
    <FormGroup label={t('settings.language.label')} helperText={t('settings.language.helper')}>
      <Select<LocaleItem>
        items={items}
        activeItem={currentLocale}
        onItemSelect={handleLocaleChange}
        itemRenderer={renderItem}
        filterable={false}
        popoverProps={{ minimal: true }}
      >
        <Button rightIcon="caret-down" icon="translate" text={currentLocale.label} />
      </Select>
    </FormGroup>
  );
}
