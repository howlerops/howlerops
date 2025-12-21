# Linting and Formatting Setup

This document describes the linting and formatting configuration for the HowlerOps frontend.

## Overview

The project uses **ESLint** for both linting and auto-formatting, configured to:
- Remove unused variables and imports automatically
- Sort imports (standard lib → third-party → local)
- Enforce TypeScript and React best practices
- Auto-fix issues on save in VSCode

## Decision: ESLint vs Biome

**Chosen: Enhanced ESLint setup**

### Rationale

We chose to enhance the existing ESLint setup rather than switch to Biome because:

1. **Ruthless Simplicity**: Project already had working ESLint configuration
2. **Minimal Delta**: Only needed to add import sorting plugin
3. **Low Risk**: No migration needed, just enhancement
4. **Philosophy Alignment**: "Start minimal, grow as needed" - we had a working base

### What We Added

- `eslint-plugin-simple-import-sort` for automatic import sorting
- VSCode settings for format-on-save
- Makefile for convenient commands

## Features

### Auto-Fix on Save (VSCode)

The `.vscode/settings.json` configures VSCode to:
- Auto-fix all ESLint issues on save
- Sort imports automatically
- Remove unused imports
- Add trailing newlines (project requirement)

### Import Sorting

Imports are automatically sorted into groups:
1. Third-party packages (e.g., `react`, `@tanstack/react-query`)
2. Local imports (e.g., `./components`, `@/lib`)

Within each group, imports are alphabetically sorted.

Example:
```typescript
// Before
import { useState } from 'react'
import { Button } from './components/ui/button'
import { useQuery } from '@tanstack/react-query'
import { formatDate } from '@/lib/utils'

// After
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'

import { Button } from './components/ui/button'
import { formatDate } from '@/lib/utils'
```

### Unused Code Removal

The `eslint-plugin-unused-imports` plugin automatically removes:
- Unused imports
- Unused variables (except those prefixed with `_`)

### Best Practices Enforcement

ESLint enforces:
- TypeScript best practices
- React hooks rules
- React refresh rules
- No constant conditions
- And more...

## Usage

### Command Line

```bash
# Check for issues
make lint

# Auto-fix all issues
make format

# Run all checks (lint + typecheck)
make check
```

Or using npm scripts directly:
```bash
npm run lint        # Check only
npm run lint:fix    # Auto-fix
```

### VSCode

Just save a file - VSCode will automatically:
1. Remove unused imports
2. Sort imports
3. Fix other ESLint issues
4. Add trailing newline

No manual action needed!

## Configuration Files

### `/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/eslint.config.js`

Main ESLint configuration using the flat config format (ESLint 9+).

Key plugins:
- `typescript-eslint` - TypeScript support
- `eslint-plugin-unused-imports` - Remove unused code
- `eslint-plugin-simple-import-sort` - Sort imports
- `eslint-plugin-react-hooks` - React hooks rules
- `eslint-plugin-react-refresh` - React refresh rules

### `/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/.vscode/settings.json`

VSCode workspace settings that enable:
- Format on save
- ESLint as default formatter
- Auto-fix on save
- Trailing newline enforcement

### `/Users/jacob/projects/amplifier/ai_working/howlerops/frontend/Makefile`

Convenient command shortcuts for common development tasks.

## Customization

### Ignoring Files

Edit the `ignores` section in `eslint.config.js`:

```javascript
{
  ignores: ['dist', 'wailsjs', 'src/lib/sync/INTEGRATION_EXAMPLE.tsx']
}
```

### Adjusting Rules

Edit the `rules` section in `eslint.config.js`:

```javascript
rules: {
  'simple-import-sort/imports': 'warn',  // Change to 'error' to make blocking
  'unused-imports/no-unused-imports': 'warn',
  // Add more rules...
}
```

### File-Specific Rules

The config already has examples of file-specific rule overrides:

```javascript
{
  files: ['**/*.test.ts', '**/*.test.tsx'],
  rules: {
    '@typescript-eslint/no-explicit-any': 'off',
    'unused-imports/no-unused-vars': 'off',
  }
}
```

## Current Status

After initial setup and auto-fix:
- All imports sorted correctly
- Unused imports removed
- ~38 warnings remaining (mostly `any` type usage - intentional warnings)
- 0 import-related errors
- Format-on-save working in VSCode

## Philosophy Alignment

This setup follows the project's "ruthless simplicity" philosophy:

✅ **Minimal configuration** - Enhanced existing setup instead of adding new tools
✅ **Direct integration** - Using ESLint's built-in capabilities
✅ **Avoid future-proofing** - Only added what's needed now
✅ **Start minimal, grow as needed** - Built on existing working base
✅ **Code you don't write has no bugs** - Auto-removal of unused code

## Future Enhancements

If needed later, consider:
- Stricter TypeScript rules (type-aware linting)
- Additional React-specific rules
- Custom import sort groups for domain-specific ordering
- Prettier integration (if team prefers separate formatter)

For now, the current setup handles all requirements efficiently.
