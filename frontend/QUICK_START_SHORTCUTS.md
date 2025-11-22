# Quick Start: Keyboard Shortcuts

## 5-Minute Integration Guide

### Step 1: Import the Provider

In `src/app.tsx`, add the import:

```tsx
import { KeyboardShortcutsProvider } from '@/components/keyboard-shortcuts-provider'
```

### Step 2: Add State Management

In your main app or dashboard component:

```tsx
// Add these hooks at the top of your component
const [aiChatOpen, setAIChatOpen] = useState(false)
const { createTab, closeTab, setActiveTab, tabs, activeTabId } = useQueryStore()
```

### Step 3: Wrap Your App

Wrap your `NavigationProvider` or main content with `KeyboardShortcutsProvider`:

```tsx
<KeyboardShortcutsProvider
  onOpenAIChat={() => setAIChatOpen(true)}
  onNewTab={() => createTab()}
  onCloseTab={() => activeTabId && closeTab(activeTabId)}
  onSwitchTab={(index) => {
    const tabIds = Object.keys(tabs)
    if (tabIds[index]) setActiveTab(tabIds[index])
  }}
>
  <NavigationProvider>
    {/* Your existing app content */}
  </NavigationProvider>
</KeyboardShortcutsProvider>
```

### Step 4: Try It Out!

1. Press `?` - See all available shortcuts
2. Press `Cmd/Ctrl+K` - Open command palette
3. Press `Cmd/Ctrl+N` - Create new tab
4. Press `Cmd/Ctrl+1` - Switch to first tab

## Available Shortcuts

### Most Important

- `Cmd/Ctrl+K` - **Command Palette** (Quick access to everything)
- `?` - **Show Help** (See all shortcuts)

### AI Features

- `Cmd/Ctrl+/` - Toggle AI Chat
- `Cmd/Ctrl+Shift+E` - Explain SQL
- `Cmd/Ctrl+Shift+F` - Fix SQL

### Navigation

- `Cmd/Ctrl+N` - New Tab
- `Cmd/Ctrl+W` - Close Tab
- `Cmd/Ctrl+1-9` - Switch Tabs

### Query

- `Cmd/Ctrl+Enter` - Execute Query

## Full Documentation

See `KEYBOARD_SHORTCUTS.md` for complete documentation.

See `KEYBOARD_SHORTCUTS_IMPLEMENTATION.md` for implementation details.

## Troubleshooting

**Shortcuts not working?**
- Make sure `KeyboardShortcutsProvider` wraps your app
- Check browser console for errors
- Press `?` to see if shortcuts are registered

**Command palette not opening?**
- Try pressing `Cmd+K` on Mac or `Ctrl+K` on Windows/Linux
- Check if another app is intercepting the shortcut

## Next Steps

1. Integrate the provider into `app.tsx`
2. Connect handlers to your existing features
3. Test all shortcuts
4. Customize as needed

The system is fully functional and ready to use!
