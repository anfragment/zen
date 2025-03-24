import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { detectSystemLanguage } from '../utils/detectSystemLanguage';

import enTranslation from './locales/en.json';
import ruTranslation from './locales/ru.json';

const initialLanguage = detectSystemLanguage();

i18n.use(initReactI18next).init({
  resources: {
    en: {
      translation: enTranslation,
    },
    ru: {
      translation: ruTranslation,
    },
  },
  lng: initialLanguage,
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

export default i18n;
