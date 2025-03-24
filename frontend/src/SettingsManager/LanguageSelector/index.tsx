import { Button, FormGroup, MenuItem } from '@blueprintjs/core';
import { ItemRenderer, Select } from '@blueprintjs/select';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

interface LanguageItem {
  value: string;
  label: string;
}

export function LanguageSelector() {
  const { t, i18n } = useTranslation();
  const [language, setLanguage] = useState<string>(localStorage.getItem('language') || 'en');

  useEffect(() => {
    const savedLanguage = localStorage.getItem('language');
    if (savedLanguage) {
      i18n.changeLanguage(savedLanguage);
      setLanguage(savedLanguage);
    }
  }, [i18n]);

  const handleLanguageChange = (item: LanguageItem): void => {
    i18n.changeLanguage(item.value);
    localStorage.setItem('language', item.value);
    setLanguage(item.value);
  };

  const items: LanguageItem[] = [
    { value: 'en', label: 'English' },
    { value: 'ru', label: 'Русский' },
  ];

  const renderItem: ItemRenderer<LanguageItem> = (item, { handleClick, handleFocus, modifiers }) => {
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

  const currentLanguage = items.find((item) => item.value === language) || items[0];

  return (
    <FormGroup label={t('settings.language.label')} helperText={t('settings.language.helper')}>
      <Select<LanguageItem>
        items={items}
        activeItem={currentLanguage}
        onItemSelect={handleLanguageChange}
        itemRenderer={renderItem}
        filterable={false}
        popoverProps={{ minimal: true }}
      >
        <Button rightIcon="caret-down" icon="translate" text={currentLanguage.label} />
      </Select>
    </FormGroup>
  );
}
