// ESLint configuration for the frontend workspace (flat config).
// Type-aware typescript-eslint rules, React hooks rules, and project rules from AGENTS.md §8.
import js from '@eslint/js';
import { defineConfig, globalIgnores } from 'eslint/config';
import reactHooks from 'eslint-plugin-react-hooks';
import tseslint from 'typescript-eslint';

const NO_NETWORK =
  'The frontend makes no network calls until a separately approved, session-authenticated portal API contract exists (owner decision, Phase 3 Group 5).';
const NO_PERSISTENCE =
  'No browser persistence: session state lives in server-managed HttpOnly cookies (AGENTS.md §8.10).';

const restrictedGlobals = [
  ...['fetch', 'XMLHttpRequest', 'WebSocket', 'EventSource'].map((name) => ({
    name,
    message: NO_NETWORK,
  })),
  ...['localStorage', 'sessionStorage', 'indexedDB', 'caches'].map((name) => ({
    name,
    message: NO_PERSISTENCE,
  })),
];

const restrictedProperties = [
  ...['window', 'globalThis', 'self'].flatMap((object) =>
    restrictedGlobals.map(({ name, message }) => ({ object, property: name, message })),
  ),
  { object: 'navigator', property: 'sendBeacon', message: NO_NETWORK },
  { object: 'navigator', property: 'serviceWorker', message: NO_PERSISTENCE },
  { object: 'document', property: 'cookie', message: NO_PERSISTENCE },
];

export default defineConfig([
  globalIgnores(['**/dist/', '**/coverage/', '**/node_modules/']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.strictTypeChecked,
      tseslint.configs.stylisticTypeChecked,
      reactHooks.configs.flat['recommended-latest'],
    ],
    languageOptions: {
      parserOptions: { projectService: true, tsconfigRootDir: import.meta.dirname },
    },
    rules: {
      'no-restricted-globals': ['error', ...restrictedGlobals],
      'no-restricted-properties': ['error', ...restrictedProperties],
      'no-restricted-syntax': [
        'error',
        {
          selector: "JSXAttribute[name.name='dangerouslySetInnerHTML']",
          message: 'Never render content as HTML (AGENTS.md §8.10).',
        },
        {
          selector: "JSXAttribute[name.name='style']",
          message:
            'Use semantic Tailwind token classes. Inline style is only for genuinely dynamic values (AGENTS.md §8.3); disable this rule on that line with a reason.',
        },
      ],
      '@typescript-eslint/consistent-type-definitions': ['error', 'type'],
    },
  },
  {
    // Node.js scripts and configuration written in JavaScript.
    files: ['**/*.{js,mjs}'],
    extends: [js.configs.recommended],
    languageOptions: { globals: { console: 'readonly', process: 'readonly' } },
  },
]);
