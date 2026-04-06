// Compatibility shim: keep legacy ".../app" imports on the runtime-safe dispatcher.
// This avoids loading generated service modules at import time in browser/Vitest mode.

export * from "./app.ts";
