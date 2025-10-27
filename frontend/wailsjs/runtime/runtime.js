/**
 * Mock Wails runtime for web deployment
 * These stubs allow the code to build in Vercel while maintaining desktop app compatibility
 */

const warn = (name) => console.warn(`Desktop-only feature: ${name}() is not available in web version`);

export function LogPrint(message) { console.log(message); }
export function LogTrace(message) { console.trace(message); }
export function LogDebug(message) { console.debug(message); }
export function LogInfo(message) { console.info(message); }
export function LogWarning(message) { console.warn(message); }
export function LogError(message) { console.error(message); }
export function LogFatal(message) { console.error(message); }

export function EventsOnMultiple(eventName, callback, maxCallbacks) {
    warn('EventsOnMultiple');
    return () => {};
}
export function EventsOn(eventName, callback) {
    warn('EventsOn');
    return () => {};
}
export function EventsOff(eventName, ...additionalEventNames) { warn('EventsOff'); }
export function EventsOnce(eventName, callback) {
    warn('EventsOnce');
    return () => {};
}
export function EventsEmit(eventName) { warn('EventsEmit'); }

export function WindowReload() { window.location.reload(); }
export function WindowReloadApp() { window.location.reload(); }
export function WindowSetAlwaysOnTop(b) { warn('WindowSetAlwaysOnTop'); }
export function WindowSetSystemDefaultTheme() { warn('WindowSetSystemDefaultTheme'); }
export function WindowSetLightTheme() { warn('WindowSetLightTheme'); }
export function WindowSetDarkTheme() { warn('WindowSetDarkTheme'); }
export function WindowCenter() { warn('WindowCenter'); }
export function WindowSetTitle(title) { document.title = title; }
export function WindowFullscreen() { warn('WindowFullscreen'); }
export function WindowUnfullscreen() { warn('WindowUnfullscreen'); }
export function WindowIsFullscreen() { return false; }
export function WindowGetSize() { return { width: window.innerWidth, height: window.innerHeight }; }
export function WindowSetSize(width, height) { warn('WindowSetSize'); }
export function WindowSetMaxSize(width, height) { warn('WindowSetMaxSize'); }
export function WindowSetMinSize(width, height) { warn('WindowSetMinSize'); }
export function WindowSetPosition(x, y) { warn('WindowSetPosition'); }
export function WindowGetPosition() { return { x: 0, y: 0 }; }
export function WindowHide() { warn('WindowHide'); }
export function WindowShow() { warn('WindowShow'); }
export function WindowMaximise() { warn('WindowMaximise'); }
export function WindowToggleMaximise() { warn('WindowToggleMaximise'); }
export function WindowUnmaximise() { warn('WindowUnmaximise'); }
export function WindowIsMaximised() { return false; }
export function WindowMinimise() { warn('WindowMinimise'); }
export function WindowUnminimise() { warn('WindowUnminimise'); }
export function WindowSetBackgroundColour(R, G, B, A) { warn('WindowSetBackgroundColour'); }
export function ScreenGetAll() { return []; }
export function WindowIsMinimised() { return false; }
export function WindowIsNormal() { return true; }
export function BrowserOpenURL(url) { window.open(url, '_blank'); }
export function Environment() { return { platform: 'web', buildType: 'production' }; }
export function Quit() { warn('Quit'); }
export function Hide() { warn('Hide'); }
export function Show() { warn('Show'); }
export function ClipboardGetText() {
    warn('ClipboardGetText - use navigator.clipboard instead');
    return Promise.resolve('');
}
export function ClipboardSetText(text) {
    warn('ClipboardSetText - use navigator.clipboard instead');
    return Promise.resolve();
}
export function OnFileDrop(callback, useDropTarget) {
    warn('OnFileDrop');
    return () => {};
}
export function OnFileDropOff() { warn('OnFileDropOff'); }
export function CanResolveFilePaths() { return false; }
export function ResolveFilePaths(files) { return Promise.resolve([]); }
