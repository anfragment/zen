import { isEmojiSupported } from 'is-emoji-supported';
import React from 'react';
import { createRoot } from 'react-dom/client';

import { GetLocale } from '../wailsjs/go/cfg/Config';

import App from './App';
import ErrorBoundary from './ErrorBoundary';
import { FALLBACK_LANG, initI18n } from './i18n';
import './style.css';

(function polyfillCountryFlagEmojis() {
  if (!isEmojiSupported('😊') || isEmojiSupported('🇨🇭')) {
    return;
  }

  const style = document.createElement('style');
  style.innerHTML = `
      body, html {
        font-family: 'Twemoji Country Flags', Inter, Roboto, 'Helvetica Neue', 'Arial Nova', 'Nimbus Sans', Arial, sans-serif;
      }
    `;
  document.head.appendChild(style);
})();

async function bootstrap() {
  let locale = FALLBACK_LANG;
  try {
    locale = await GetLocale();
  } catch {
    console.warn('Failed to fetch locale, falling back to "en"');
  }

  await initI18n(locale);

  const container = document.getElementById('root');
  const root = createRoot(container!);

  root.render(
    <React.StrictMode>
      <ErrorBoundary>
        <App />
      </ErrorBoundary>
    </React.StrictMode>,
  );
}

bootstrap();
