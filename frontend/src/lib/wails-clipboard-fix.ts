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
      } catch {
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
        } catch {
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
      } catch {
        // Return fallback clipboard content
        return fallbackClipboard;
      }
      return '';
    };

    // Override write (for rich content)
    if (nativeWrite) {
      window.navigator.clipboard.write = async function(data: ClipboardItems): Promise<void> {
        try {
          await nativeWrite(data);
        } catch {
          // Silently fail for rich content
        }
      };
    }

    // Override read
    if (nativeRead) {
      window.navigator.clipboard.read = async function(): Promise<ClipboardItems> {
        try {
          return await nativeRead();
        } catch {
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
      } catch {
        // Fail silently
        return false;
      }
    }
    // Allow other commands to proceed normally
    return originalExecCommand.call(document, command, showUI, value);
  };

  // Fix 3: Add global error handler to catch any remaining clipboard errors
  const originalConsoleError = console.error;
  console.error = function(...args: unknown[]) {
    // Filter out clipboard permission errors ONLY
    const errorString = args.join(' ');
    const isClipboardError = errorString.includes('NotAllowedError') &&
        (errorString.includes('clipboard') || errorString.includes('clipboardService'));
    
    if (isClipboardError) {
      // Silently ignore clipboard errors in WAILS
      return;
    }
    
    // Pass through ALL other errors unchanged
    originalConsoleError.apply(console, args);
  };

  // Fix 4: Monaco clipboard service patch
  // Note: Monaco EditorOptions are read-only and modifying them causes infinite reloads.
  // The clipboard API overrides above are sufficient for handling clipboard operations.
  // Monaco will gracefully handle clipboard failures through our overridden APIs.

  console.debug('WAILS clipboard fixes applied successfully');
}