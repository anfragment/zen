import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { GetLocale, SetLocale } from '../../wailsjs/go/cfg/Config';

import deDE from './locales/de-DE.json';
import enUS from './locales/en-US.json';
import kkKZ from './locales/kk-KZ.json';
import ruRU from './locales/ru-RU.json';

export const SUPPORTED_LOCALES = ['en', 'en-US', 'de', 'de-DE', 'kk', 'kk-KZ', 'ru', 'ru-RU'] as const;
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];
export const FALLBACK_LOCALE: SupportedLocale = 'en-US';

export function detectSystemLocale(): SupportedLocale {
  const browserLang = navigator.language;
  const detected = SUPPORTED_LOCALES.includes(browserLang as any) ? (browserLang as SupportedLocale) : FALLBACK_LOCALE;

  return detected;
}

export function getCurrentLocale(): SupportedLocale {
  return (i18n.language as SupportedLocale) || FALLBACK_LOCALE;
}

export async function changeLocale(locale: SupportedLocale) {
  const normalized = SUPPORTED_LOCALES.includes(locale) ? locale : FALLBACK_LOCALE;
  await i18n.changeLanguage(normalized);
  await SetLocale(normalized);
}

export async function initI18n() {
  let locale = await GetLocale();
  if (locale === '') {
    const detected = detectSystemLocale();
    await SetLocale(detected);
    locale = detected;
  }

  return i18n.use(initReactI18next).init({
    resources: {
      en: { translation: enUS },
      'en-US': { translation: enUS },
      de: { translation: deDE },
      'de-DE': { translation: deDE },
      kk: { translation: kkKZ },
      'kk-KZ': { translation: kkKZ },
      ru: { translation: ruRU },
      'ru-RU': { translation: ruRU },
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
