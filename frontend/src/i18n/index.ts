import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import enTranslation from './locales/en.json';
import ruTranslation from './locales/ru.json';

export async function initI18n(locale: string) {
  return i18n.use(initReactI18next).init({
    resources: {
      en: { translation: enTranslation },
      ru: { translation: ruTranslation },
    },
    lng: locale,
    fallbackLng: 'en',
    returnNull: false,
    returnEmptyString: false,
    interpolation: {
      escapeValue: false,
    },
    react: {
      useSuspense: false,
    },
  });
}
