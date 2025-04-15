export default {
  // Where to look for keys
  input: ['src/**/*.{js,jsx,ts,tsx}'],

  // Where to write the translations
  output: 'src/i18n/locales/$LOCALE.json',

  // Your languages
  locales: ['en-US', 'de-DE', 'kk-KZ', 'ru-RU'],

  // Set to false to disable saving old catalogs
  createOldCatalogs: false,

  // Use nested JSON
  namespaceSeparator: false,
  keySeparator: '.',

  // Sort keys alphabetically
  sort: true,

  // Don't add empty translations for every locale - just add the keys
  // This preserves existing translations
  keepRemoved: true,

  // Add source file references (helps for debugging)
  addTags: false,

  // Functions to look for
  lexers: {
    js: [
      {
        lexer: 'JsxLexer',
        functions: ['t'], // Look for t() function
        namespaceFunctions: ['useTranslation', 'withTranslation'], // Also track these
      },
    ],
    jsx: [
      {
        lexer: 'JsxLexer',
        functions: ['t'],
        namespaceFunctions: ['useTranslation', 'withTranslation'],
      },
    ],
    ts: [
      {
        lexer: 'JsxLexer',
        functions: ['t'],
        namespaceFunctions: ['useTranslation', 'withTranslation'],
      },
    ],
    tsx: [
      {
        lexer: 'JsxLexer',
        functions: ['t'],
        namespaceFunctions: ['useTranslation', 'withTranslation'],
      },
    ],
  },

  // Default value handling - for English, use the key itself
  defaultValue: function (locale, namespace, key) {
    if (locale === 'en_US') {
      return key;
    }
    return ''; // Empty string for other languages
  },

  // Output format
  indentation: 2,

  // Whether to add location data to JSON files
  lineEnding: 'auto',

  // Verbose output
  verbose: true,
};
