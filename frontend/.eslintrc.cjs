module.exports = {
  root: true,
  parser: "@typescript-eslint/parser",
  plugins: [
    "@typescript-eslint",
    "prettier"
  ],
  extends: [
    "airbnb",
    "airbnb-typescript",
    "plugin:import/typescript",
    "plugin:prettier/recommended"
  ],
  rules: {
    "no-console": 1,
    "react/react-in-jsx-scope": 0,
    "import/order": ["error", {
      "newlines-between": "always",
      "groups": ["builtin", "external", "internal", "parent", "sibling", "index"],
      "alphabetize": {
        "order": "asc",
        "caseInsensitive": true
      }
    }],
    "import/prefer-default-export": 0,
    "@typescript-eslint/no-use-before-define": 0,
    "@typescript-eslint/no-shadow": 0,
    "react/no-unstable-nested-components": 0,
    "react/require-default-props": 0,
    "import/no-relative-packages": 0,
    "no-console": 0
  },
  parserOptions: {
    project: "./tsconfig.json",
    tsconfigRootDir: __dirname
  },
  ignorePatterns: [
    "node_modules/",
    "wailsjs/",
    "vite.config.ts"
  ]
};
