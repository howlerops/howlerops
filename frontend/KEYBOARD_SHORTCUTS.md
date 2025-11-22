# Keyboard Shortcuts Documentation

## Overview

HowlerOps includes a comprehensive keyboard shortcuts system for power users, featuring:

- **Global keyboard shortcuts** for common actions
- **Command palette** (Cmd/Ctrl+K) for fuzzy search of all commands
- **Context-aware shortcuts** that adapt to the current view
- **Visual help overlay** (press `?`) showing all available shortcuts
- **Customizable shortcuts** (coming soon)

## Quick Reference

Press `?` anywhere in the app to see all available shortcuts.

Press `Cmd/Ctrl+K` to open the command palette for quick access to any action.

## Keyboard Shortcuts by Category

### AI Features

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Cmd/Ctrl+/` | Toggle AI Chat | Open or close the AI chat sidebar |
| `Cmd/Ctrl+Shift+E` | Explain Selected SQL | Get AI explanation of highlighted SQL |
| `Cmd/Ctrl+Shift+F` | Fix Selected SQL | Let AI fix errors in highlighted SQL |

### Navigation

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Cmd/Ctrl+N` | New Query Tab | Create a new query editor tab |
| `Cmd/Ctrl+W` | Close Tab | Close the current tab |
| `Cmd/Ctrl+1-9` | Switch to Tab 1-9 | Activate query tab by number |

### Query Execution

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Cmd/Ctrl+Enter` | Execute Query | Run the current SQL query |
| `Cmd/Ctrl+Shift+Enter` | Execute and Explain | Run query and get AI explanation |

### General

| Shortcut | Action | Description |
|----------|--------|-------------|
| `Cmd/Ctrl+K` | Command Palette | Open command palette for quick actions |
| `?` | Keyboard Help | Show all available shortcuts |
| `Esc` | Close Dialog | Close the current dialog or overlay |

## Command Palette

The command palette provides fuzzy search across all available actions. It supports:

- **Fuzzy matching** - Type partial words to find commands
- **Keyboard navigation** - Use arrow keys to navigate, Enter to execute
- **Recent commands** - Shows your last 5 commands for quick access
- **Grouped by category** - AI, Database, Settings, Navigation
- **Visual feedback** - See which commands are available

### Command Palette Shortcuts

| Shortcut | Action |
|----------|--------|
| `↑↓` | Navigate commands |
| `Enter` | Execute selected command |
| `Esc` | Close palette |

## Implementation Details

### Architecture

The keyboard shortcuts system consists of:

1. **Types** (`types/keyboard-shortcuts.ts`)
   - `KeyboardShortcut` - Individual shortcut definition
   - `ShortcutCategory` - Grouped shortcuts
   - Helper functions for formatting and matching

2. **Hooks**
   - `useKeyboardShortcuts` - Core shortcuts management
   - `useCommandPalette` - Command palette state and actions

3. **Components**
   - `KeyboardShortcutsHelper` - Visual help overlay
   - `CommandPalette` - Command palette UI
   - `KeyboardShortcutsProvider` - Global provider

4. **Provider** (`components/keyboard-shortcuts-provider.tsx`)
   - Wraps the app to provide global shortcuts
   - Registers all shortcuts on mount
   - Manages help and command palette state

### Integration

To use the keyboard shortcuts system:

```tsx
import { KeyboardShortcutsProvider } from '@/components/keyboard-shortcuts-provider'

<KeyboardShortcutsProvider
  onGenerateSQL={() => { /* ... */ }}
  onOpenAIChat={() => { /* ... */ }}
  onNewTab={() => { /* ... */ }}
  // ... other handlers
>
  <App />
</KeyboardShortcutsProvider>
```

### Context-Aware Behavior

Shortcuts are context-aware and will:

- **Ignore input fields** - Don't trigger when typing in inputs/textareas
- **Allow Cmd/Ctrl+Enter in textareas** - Common pattern for sending messages
- **Prevent browser conflicts** - Use preventDefault on matched shortcuts
- **Stop propagation** - Prevent multiple handlers from firing

### Customization

To add custom shortcuts:

```tsx
import { useKeyboardShortcutsContext } from '@/components/keyboard-shortcuts-provider'

function MyComponent() {
  const { registerShortcut, unregisterShortcut } = useKeyboardShortcutsContext()

  useEffect(() => {
    registerShortcut({
      id: 'my-custom-shortcut',
      key: 's',
      label: 'Save',
      description: 'Save current work',
      category: 'general',
      modifiers: ['cmd'],
      handler: () => console.log('Save!'),
    })

    return () => unregisterShortcut('my-custom-shortcut')
  }, [])
}
```

## Platform Differences

The system automatically detects the platform and adjusts:

- **macOS**: Uses Cmd (⌘) key, shows Mac symbols (⌘, ⌥, ⇧)
- **Windows/Linux**: Uses Ctrl key, shows text modifiers (Ctrl+, Alt+, Shift+)

## Accessibility

- All shortcuts include ARIA labels for screen readers
- Visual help overlay can be accessed via keyboard (`?`)
- Command palette is fully keyboard navigable
- Shortcuts don't interfere with native browser accessibility features

## Future Enhancements

- [ ] User-customizable shortcuts via settings
- [ ] Export/import shortcut configurations
- [ ] Shortcut recording UI
- [ ] Conflict detection and warnings
- [ ] Per-view shortcut contexts
- [ ] Shortcut chaining (multi-key sequences)

## Troubleshooting

### Shortcuts Not Working

1. **Check if input is focused** - Shortcuts are disabled in input fields
2. **Check browser conflicts** - Some shortcuts may conflict with browser shortcuts
3. **Open help overlay** - Press `?` to see if shortcut is registered
4. **Check console** - Look for keyboard shortcut errors

### Command Palette Not Opening

1. **Verify Cmd/Ctrl+K** - Make sure you're using the correct modifier key
2. **Check for conflicts** - Some apps intercept Cmd/Ctrl+K
3. **Try clicking** - Use the UI button as fallback

### Performance Issues

The keyboard shortcuts system is optimized for performance:

- Uses event delegation for efficiency
- Shortcuts are registered once on mount
- Minimal re-renders with React Context
- Fuzzy search is debounced

## Examples

See `components/keyboard-shortcuts-integration-example.tsx` for complete integration examples.

## API Reference

### KeyboardShortcutsProvider Props

```typescript
interface KeyboardShortcutsProviderProps {
  children: React.ReactNode
  onGenerateSQL?: () => void
  onExplainQuery?: () => void
  onFixSQL?: () => void
  onOptimizeQuery?: () => void
  onOpenAIChat?: () => void
  onAddConnection?: () => void
  onSwitchDatabase?: () => void
  onOpenSettings?: () => void
  onNewTab?: () => void
  onCloseTab?: () => void
  onSwitchTab?: (index: number) => void
  onExecuteQuery?: () => void
  onExecuteAndExplain?: () => void
}
```

### useKeyboardShortcutsContext

```typescript
interface KeyboardShortcutsContextValue {
  registerShortcut: (shortcut: KeyboardShortcut) => void
  unregisterShortcut: (id: string) => void
  showHelp: () => void
  showCommandPalette: () => void
}
```

## Contributing

To add new shortcuts:

1. Add the handler prop to `KeyboardShortcutsProvider`
2. Register the shortcut in the provider's `useEffect`
3. Add the command action to `useCommandPalette`
4. Update this documentation
5. Add tests for the new shortcut

## License

Part of HowlerOps, all rights reserved.
