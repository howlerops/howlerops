/**
 * WAILS WebView Clipboard Fix
 *
 * This module provides a comprehensive fix for clipboard permission errors
 * that occur when Monaco Editor runs inside a WAILS WebView environment.
 *
 * The issue occurs because:
 * 1. WAILS WebView has restricted clipboard access
 * 2. Monaco Editor attempts to access clipboard for various features
 * 3. These attempts fail with NotAllowedError permissions errors
 *
 * This fix intercepts and handles these errors gracefully.
 */

declare global {
  interface Window {
    __wailsClipboardFixed?: boolean;
  }
}

/**
 * Apply comprehensive clipboard fixes for WAILS WebView
 */
export function applyWailsClipboardFix() {
  // Prevent multiple applications
  if (window.__wailsClipboardFixed) {
    return;
  }

  // Mark as fixed
  window.__wailsClipboardFixed = true;

  // Fix 1: Override clipboard API with safe wrappers
  if (window.navigator?.clipboard) {
    const originalClipboard = window.navigator.clipboard;
    const nativeWriteText = originalClipboard.writeText?.bind(originalClipboard);
    const nativeReadText = originalClipboard.readText?.bind(originalClipboard);
    const nativeWrite = originalClipboard.write?.bind(originalClipboard);
    const nativeRead = originalClipboard.read?.bind(originalClipboard);

    // Create a fallback clipboard storage
    let fallbackClipboard = '';

    // Override writeText
    window.navigator.clipboard.writeText = async function(text: string): Promise<void> {
      try {
        if (nativeWriteText) {
          await nativeWriteText(text);
        }
      } catch (err) {
        // Use fallback storage
        fallbackClipboard = text;
        // Try legacy method as fallback
        try {
          const textArea = document.createElement('textarea');
          textArea.value = text;
          textArea.style.position = 'fixed';
          textArea.style.left = '-999999px';
          document.body.appendChild(textArea);
          textArea.select();
          document.execCommand('copy');
          document.body.removeChild(textArea);
        } catch (e) {
          // Silently fail
        }
      }
    };

    // Override readText
    window.navigator.clipboard.readText = async function(): Promise<string> {
      try {
        if (nativeReadText) {
          return await nativeReadText();
        }
      } catch (err) {
        // Return fallback clipboard content
        return fallbackClipboard;
      }
      return '';
    };

    // Override write (for rich content)
    if (originalClipboard.write) {
      window.navigator.clipboard.write = async function(data: ClipboardItems): Promise<void> {
        try {
          if (nativeWrite) {
            await nativeWrite(data);
          }
        } catch (err) {
          // Silently fail for rich content
        }
      };
    }

    // Override read
    if (originalClipboard.read) {
      window.navigator.clipboard.read = async function(): Promise<ClipboardItems> {
        try {
          if (nativeRead) {
            return await nativeRead();
          }
        } catch (err) {
          // Return empty clipboard items
          return [];
        }
        return [];
      };
    }
  }

  // Fix 2: Override document.execCommand for clipboard operations
  const originalExecCommand = document.execCommand;
  document.execCommand = function(command: string, showUI?: boolean, value?: string): boolean {
    // Intercept clipboard commands
    if (command === 'copy' || command === 'cut' || command === 'paste') {
      try {
        return originalExecCommand.call(document, command, showUI, value);
      } catch (err) {
        // Fail silently
        return false;
      }
    }
    // Allow other commands to proceed normally
    return originalExecCommand.call(document, command, showUI, value);
  };

  // Fix 3: Add global error handler to catch any remaining clipboard errors
  const originalConsoleError = console.error;
  console.error = function(...args: any[]) {
    // Filter out clipboard permission errors
    const errorString = args.join(' ');
    if (errorString.includes('NotAllowedError') &&
        (errorString.includes('clipboard') || errorString.includes('clipboardService'))) {
      // Log as debug instead of error
      console.debug('Clipboard access blocked (expected in WAILS):', ...args);
      return;
    }
    // Pass through other errors
    originalConsoleError.apply(console, args);
  };

  // Fix 4: Patch Monaco's clipboard service if it's loaded
  if ((window as any).monaco?.editor) {
    const monaco = (window as any).monaco;

    // Disable clipboard features in Monaco defaults
    if (monaco.editor.EditorOptions) {
      monaco.editor.EditorOptions.copyWithSyntaxHighlighting = { defaultValue: false };
      monaco.editor.EditorOptions.emptySelectionClipboard = { defaultValue: false };
    }
  }

  console.debug('WAILS clipboard fixes applied successfully');
}