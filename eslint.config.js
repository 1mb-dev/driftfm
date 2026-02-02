export default [
  {
    files: ['web/**/*.js'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: {
        // Browser globals
        window: 'readonly',
        document: 'readonly',
        navigator: 'readonly',
        localStorage: 'readonly',
        console: 'readonly',
        setTimeout: 'readonly',
        clearTimeout: 'readonly',
        setInterval: 'readonly',
        clearInterval: 'readonly',
        fetch: 'readonly',
        URL: 'readonly',
        URLSearchParams: 'readonly',
        HTMLElement: 'readonly',
        MediaMetadata: 'readonly',
        Audio: 'readonly',
        history: 'readonly',
        location: 'readonly',
      }
    },
    rules: {
      // Errors — catch real bugs
      'no-undef': 'error',
      'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      'no-constant-condition': 'error',
      'no-dupe-keys': 'error',
      'no-duplicate-case': 'error',
      'no-empty': ['error', { allowEmptyCatch: false }],
      'no-unreachable': 'error',
      'no-unsafe-negation': 'error',
      'use-isnan': 'error',
      'valid-typeof': 'error',
      'no-self-assign': 'error',
      'no-self-compare': 'error',
      'no-template-curly-in-string': 'warn',

      // Best practices
      'eqeqeq': ['error', 'always'],
      'no-eval': 'error',
      'no-implied-eval': 'error',
      'no-throw-literal': 'error',
      'prefer-const': ['error', { destructuring: 'all' }],
      'no-var': 'error',
      'no-debugger': 'error',
    }
  },
  {
    // Test files can use test runner globals
    files: ['web/tests/**/*.js'],
    rules: {
      'no-unused-vars': 'off',
    }
  }
];
