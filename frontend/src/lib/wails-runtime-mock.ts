/**
 * Mock implementation of @wailsio/runtime for web builds (Vercel)
 *
 * This module provides stub implementations that allow the app to build
 * and run in a web context, while showing appropriate errors when
 * Wails-specific features are used.
 */

// CancellablePromise that matches the Wails v3 interface
export class CancellablePromise<T> extends Promise<T> {
  private _cancelled = false;

  constructor(
    executor: (
      resolve: (value: T | PromiseLike<T>) => void,
      reject: (reason?: unknown) => void
    ) => void
  ) {
    super(executor);
  }

  cancel(): void {
    this._cancelled = true;
  }

  get cancelled(): boolean {
    return this._cancelled;
  }
}

// Create a rejected promise for Wails calls in web mode
function createWebUnavailablePromise<T>(methodName: string): CancellablePromise<T> {
  return new CancellablePromise<T>((_, reject) => {
    reject(new Error(`${methodName} is not available in web mode. This feature requires the Wails desktop runtime.`));
  });
}

// Call mock - simulates the Wails Call.ByID function
export const Call = {
  ByID: <T>(_id: number, ..._args: unknown[]): CancellablePromise<T> => {
    return createWebUnavailablePromise<T>('Wails backend call');
  },
  ByName: <T>(_name: string, ..._args: unknown[]): CancellablePromise<T> => {
    return createWebUnavailablePromise<T>('Wails backend call');
  },
};

// Create mock - matches @wailsio/runtime/dist/create.js API exactly.
// These are used at module-load time by auto-generated bindings (models.ts).
function _Any(source: any): any {
  return source;
}

function _ByteSlice(source: any): string {
  return source == null ? "" : source;
}

function _Array(element: (source: any) => any): (source: any) => any[] {
  if (element === _Any) {
    return (source: any) => (source === null ? [] : source);
  }
  return (source: any) => {
    if (source === null) return [];
    for (let i = 0; i < source.length; i++) {
      source[i] = element(source[i]);
    }
    return source;
  };
}

function _Map(
  _key: (source: any) => string,
  value: (source: any) => any,
): (source: any) => Record<string, any> {
  if (value === _Any) {
    return (source: any) => (source === null ? {} : source);
  }
  return (source: any) => {
    if (source === null) return {};
    for (const k in source) {
      source[k] = value(source[k]);
    }
    return source;
  };
}

function _Nullable(element: (source: any) => any): (source: any) => any | null {
  if (element === _Any) return _Any;
  return (source: any) => (source === null ? null : element(source));
}

function _Struct(
  createField: Record<string, (source: any) => any>,
): (source: any) => any {
  let allAny = true;
  for (const name in createField) {
    if (createField[name] !== _Any) {
      allAny = false;
      break;
    }
  }
  if (allAny) return _Any;
  return (source: any) => {
    for (const name in createField) {
      if (name in source) {
        source[name] = createField[name](source[name]);
      }
    }
    return source;
  };
}

export const Create = {
  Any: _Any,
  ByteSlice: _ByteSlice,
  Array: _Array,
  Map: _Map,
  Nullable: _Nullable,
  Struct: _Struct,
};

// Events mock
export const Events = {
  On: (_eventName: string, _callback: (...args: unknown[]) => void): () => void => {
    console.warn('Wails events are not available in web mode');
    return () => {}; // Return unsubscribe function
  },
  Off: (_eventName: string): void => {
    // No-op in web mode
  },
  Emit: (_eventName: string, ..._args: unknown[]): void => {
    console.warn('Wails events are not available in web mode');
  },
  Once: (_eventName: string, _callback: (...args: unknown[]) => void): () => void => {
    console.warn('Wails events are not available in web mode');
    return () => {};
  },
};

// Window mock
export const Window = {
  Center: (): void => {},
  Close: (): void => {},
  Fullscreen: (): void => {},
  GetSize: (): { width: number; height: number } => ({ width: window.innerWidth, height: window.innerHeight }),
  GetPosition: (): { x: number; y: number } => ({ x: window.screenX, y: window.screenY }),
  Hide: (): void => {},
  IsFullscreen: (): boolean => false,
  IsMaximised: (): boolean => false,
  IsMinimised: (): boolean => false,
  Maximise: (): void => {},
  Minimise: (): void => {},
  SetAlwaysOnTop: (_onTop: boolean): void => {},
  SetMaxSize: (_width: number, _height: number): void => {},
  SetMinSize: (_width: number, _height: number): void => {},
  SetPosition: (_x: number, _y: number): void => {},
  SetSize: (_width: number, _height: number): void => {},
  SetTitle: (title: string): void => { document.title = title; },
  Show: (): void => {},
  UnFullscreen: (): void => {},
  UnMaximise: (): void => {},
  UnMinimise: (): void => {},
};

// Application mock
export const Application = {
  Quit: (): void => {
    console.warn('Application.Quit is not available in web mode');
  },
  Hide: (): void => {},
  Show: (): void => {},
};

// Browser mock
export const Browser = {
  OpenURL: (url: string): void => {
    window.open(url, '_blank');
  },
};

// Clipboard mock
export const Clipboard = {
  GetText: async (): Promise<string> => {
    try {
      return await navigator.clipboard.readText();
    } catch {
      console.warn('Clipboard access denied in web mode');
      return '';
    }
  },
  SetText: async (text: string): Promise<void> => {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      console.warn('Clipboard access denied in web mode');
    }
  },
};

// Screens mock
export const Screens = {
  GetAll: (): unknown[] => {
    return [{
      id: 'primary',
      name: 'Primary Display',
      width: window.screen.width,
      height: window.screen.height,
      isPrimary: true,
    }];
  },
  GetPrimary: (): unknown => {
    return {
      id: 'primary',
      name: 'Primary Display',
      width: window.screen.width,
      height: window.screen.height,
      isPrimary: true,
    };
  },
};

// System mock
export const System = {
  IsDarkMode: (): boolean => {
    return window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? false;
  },
};

// Dialogs mock
export const Dialogs = {
  Info: (_options: { title?: string; message: string }): Promise<void> => {
    return Promise.resolve();
  },
  Warning: (_options: { title?: string; message: string }): Promise<void> => {
    return Promise.resolve();
  },
  Error: (_options: { title?: string; message: string }): Promise<void> => {
    return Promise.resolve();
  },
  Question: (_options: { title?: string; message: string }): Promise<string> => {
    return Promise.resolve('No');
  },
  OpenFile: (_options?: unknown): Promise<string | null> => {
    console.warn('File dialogs are not available in web mode');
    return Promise.resolve(null);
  },
  SaveFile: (_options?: unknown): Promise<string | null> => {
    console.warn('File dialogs are not available in web mode');
    return Promise.resolve(null);
  },
};

// Log mock
export const Log = {
  Print: (message: string): void => console.log(message),
  Trace: (message: string): void => console.trace(message),
  Debug: (message: string): void => console.debug(message),
  Info: (message: string): void => console.info(message),
  Warning: (message: string): void => console.warn(message),
  Error: (message: string): void => console.error(message),
  Fatal: (message: string): void => console.error('[FATAL]', message),
};

// WML mock (Wails Markup Language)
export const WML = {
  Reload: (): void => {
    window.location.reload();
  },
};

// Flags mock
export const Flags = {
  GetAll: (): Record<string, unknown> => ({}),
  GetBoolean: (_key: string): boolean => false,
  GetString: (_key: string): string => '',
  GetInt: (_key: string): number => 0,
};

// Environment detection
export const Environment = {
  IsDesktop: (): boolean => false,
  IsWeb: (): boolean => true,
  IsMac: (): boolean => navigator.platform.toLowerCase().includes('mac'),
  IsWindows: (): boolean => navigator.platform.toLowerCase().includes('win'),
  IsLinux: (): boolean => navigator.platform.toLowerCase().includes('linux'),
};

// Default export for compatibility
export default {
  Call,
  Create,
  Events,
  Window,
  Application,
  Browser,
  Clipboard,
  Screens,
  System,
  Dialogs,
  Log,
  WML,
  Flags,
  Environment,
  CancellablePromise,
};
