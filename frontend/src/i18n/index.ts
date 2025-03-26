import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { SetLocale } from '../../wailsjs/go/cfg/Config';

import enUS from './locales/en_US.json';
import ruRU from './locales/ru_RU.json';

export const FALLBACK_LOCALE = 'en_US';
export const supportedLocales = ['en_US', 'ru_RU'];

export function detectSystemLocale(): string {
  const browserLang = navigator.language;
  const detected = supportedLocales.includes(browserLang) ? browserLang : FALLBACK_LOCALE;

  return detected;
}

export function getCurrentLocale(): string {
  return i18n.language || FALLBACK_LOCALE;
}

export async function changeLocale(locale: string) {
  const normalized = supportedLocales.includes(locale) ? locale : FALLBACK_LOCALE;
  await i18n.changeLanguage(normalized);
  await SetLocale(normalized);
}

export async function initI18n(locale: string) {
  return i18n.use(initReactI18next).init({
    resources: {
      en: { translation: enUS },
      en_US: { translation: enUS },
      ru: { translation: ruRU },
      ru_RU: { translation: ruRU },
    },
    lng: locale,
    fallbackLng: FALLBACK_LOCALE,
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
