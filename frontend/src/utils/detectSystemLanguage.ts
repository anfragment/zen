export function detectSystemLanguage(): string {
  const savedLanguage = localStorage.getItem('language');
  if (savedLanguage) {
    return savedLanguage;
  }

  const browserLang = navigator.language.split('-')[0];
  const supportedLanguages = ['en', 'ru'];
  const detectedLanguage = supportedLanguages.includes(browserLang) ? browserLang : 'en';

  localStorage.setItem('language', detectedLanguage);

  return detectedLanguage;
}

export function getSupportedLanguages(): string[] {
  return ['en', 'ru'];
}
