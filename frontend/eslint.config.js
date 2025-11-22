import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import unusedImports from 'eslint-plugin-unused-imports'
import simpleImportSort from 'eslint-plugin-simple-import-sort'

export default tseslint.config(
  {
    ignores: ['dist', 'wailsjs', 'src/lib/sync/INTEGRATION_EXAMPLE.tsx']
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      'unused-imports': unusedImports,
      'simple-import-sort': simpleImportSort,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
      // Auto-sort imports: standard lib → third-party → local
      'simple-import-sort/imports': 'warn',
      'simple-import-sort/exports': 'warn',
      // Auto-fix unused imports and variables
      'unused-imports/no-unused-imports': 'warn',
      'unused-imports/no-unused-vars': ['warn', {
        vars: 'all',
        varsIgnorePattern: '^_',
        args: 'after-used',
        argsIgnorePattern: '^_'
      }],
      // Disable the base rule as it's replaced by unused-imports
      '@typescript-eslint/no-unused-vars': 'off',
      // Global relaxations to avoid blocking on non-critical issues
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-require-imports': 'warn',
      '@typescript-eslint/ban-ts-comment': 'warn',
      'no-useless-escape': 'warn',
      // React Hooks stricter rules downgraded to warnings/off
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
      'react-hooks/purity': 'off',
      'react-hooks/immutability': 'warn',
      'react-hooks/set-state-in-effect': 'off',
      'react-hooks/preserve-manual-memoization': 'off',
      // TanStack libraries (Table, Virtual) use internal state that React Compiler cannot optimize
      'react-hooks/incompatible-library': 'off',
    },
  },
  // Relax rules for known legacy/experimental areas to avoid blocking lint
  {
    files: [
      'src/lib/sanitization/**',
      'src/lib/json-*.ts',
      'src/lib/secure-storage.ts',
      'src/lib/schema-config.ts',
      'src/lib/sync/**/*.ts',
      'src/workers/**/*.ts',
    ],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-require-imports': 'off',
      '@typescript-eslint/ban-ts-comment': 'warn',
    }
  },
  {
    files: [
      '**/*.test.ts',
      '**/*.test.tsx',
      'src/lib/sanitization/__tests__/**',
    ],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      'unused-imports/no-unused-vars': 'off',
      'no-useless-escape': 'off',
    }
  },
  {
    files: [
      'src/components/tutorials/**',
      'src/components/upgrade-*/**',
      'src/components/upgrade-*.tsx',
      'src/components/templates/**',
      'src/pages/**',
      'src/services/**',
      'src/components/organizations/**',
    ],
    rules: {
      'react-hooks/purity': 'off',
      'react-hooks/rules-of-hooks': 'warn',
      'react-hooks/exhaustive-deps': 'warn',
      '@typescript-eslint/no-explicit-any': 'warn',
    }
  },
  // UI components that export utilities alongside components
  {
    files: [
      'src/components/ui/button.tsx',
    ],
    rules: {
      'react-refresh/only-export-components': 'off', // Allow exporting buttonVariants alongside Button
    }
  },
)
