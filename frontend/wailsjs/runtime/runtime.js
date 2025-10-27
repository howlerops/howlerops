/**
 * Wails runtime wrapper - works in both desktop and web modes
 * In desktop mode: uses window.runtime provided by Wails
 * In web mode: provides safe mock implementations
 */

const isDesktop = () => typeof window !== 'undefined' && window.runtime;
const warn = (name) => console.warn(`Desktop-only feature: ${name}() is not available in web version`);

// Logging functions
export function LogPrint(message) {
    if (isDesktop()) return window.runtime.LogPrint(message);
    console.log(message);
}

export function LogTrace(message) {
    if (isDesktop()) return window.runtime.LogTrace(message);
    console.trace(message);
}

export function LogDebug(message) {
    if (isDesktop()) return window.runtime.LogDebug(message);
    console.debug(message);
}

export function LogInfo(message) {
    if (isDesktop()) return window.runtime.LogInfo(message);
    console.info(message);
}

export function LogWarning(message) {
    if (isDesktop()) return window.runtime.LogWarning(message);
    console.warn(message);
}

export function LogError(message) {
    if (isDesktop()) return window.runtime.LogError(message);
    console.error(message);
}

export function LogFatal(message) {
    if (isDesktop()) return window.runtime.LogFatal(message);
    console.error(message);
}

// Events
export function EventsOnMultiple(eventName, callback, maxCallbacks) {
    if (isDesktop()) return window.runtime.EventsOnMultiple(eventName, callback, maxCallbacks);
    warn('EventsOnMultiple');
    return () => {};
}

export function EventsOn(eventName, callback) {
    if (isDesktop()) return window.runtime.EventsOnMultiple(eventName, callback, -1);
    warn('EventsOn');
    return () => {};
}

export function EventsOff(eventName, ...additionalEventNames) {
    if (isDesktop()) return window.runtime.EventsOff(eventName, ...additionalEventNames);
    warn('EventsOff');
}

export function EventsOnce(eventName, callback) {
    if (isDesktop()) return window.runtime.EventsOnMultiple(eventName, callback, 1);
    warn('EventsOnce');
    return () => {};
}

export function EventsEmit(eventName) {
    if (isDesktop()) {
        let args = [eventName].slice.call(arguments);
        return window.runtime.EventsEmit.apply(null, args);
    }
    warn('EventsEmit');
}

// Window functions
export function WindowReload() {
    if (isDesktop()) return window.runtime.WindowReload();
    window.location.reload();
}

export function WindowReloadApp() {
    if (isDesktop()) return window.runtime.WindowReloadApp();
    window.location.reload();
}

export function WindowSetAlwaysOnTop(b) {
    if (isDesktop()) return window.runtime.WindowSetAlwaysOnTop(b);
    warn('WindowSetAlwaysOnTop');
}

export function WindowSetSystemDefaultTheme() {
    if (isDesktop()) return window.runtime.WindowSetSystemDefaultTheme();
    warn('WindowSetSystemDefaultTheme');
}

export function WindowSetLightTheme() {
    if (isDesktop()) return window.runtime.WindowSetLightTheme();
    warn('WindowSetLightTheme');
}

export function WindowSetDarkTheme() {
    if (isDesktop()) return window.runtime.WindowSetDarkTheme();
    warn('WindowSetDarkTheme');
}

export function WindowCenter() {
    if (isDesktop()) return window.runtime.WindowCenter();
    warn('WindowCenter');
}

export function WindowSetTitle(title) {
    if (isDesktop()) return window.runtime.WindowSetTitle(title);
    document.title = title;
}

export function WindowFullscreen() {
    if (isDesktop()) return window.runtime.WindowFullscreen();
    warn('WindowFullscreen');
}

export function WindowUnfullscreen() {
    if (isDesktop()) return window.runtime.WindowUnfullscreen();
    warn('WindowUnfullscreen');
}

export function WindowIsFullscreen() {
    if (isDesktop()) return window.runtime.WindowIsFullscreen();
    return false;
}

export function WindowGetSize() {
    if (isDesktop()) return window.runtime.WindowGetSize();
    return { width: window.innerWidth, height: window.innerHeight };
}

export function WindowSetSize(width, height) {
    if (isDesktop()) return window.runtime.WindowSetSize(width, height);
    warn('WindowSetSize');
}

export function WindowSetMaxSize(width, height) {
    if (isDesktop()) return window.runtime.WindowSetMaxSize(width, height);
    warn('WindowSetMaxSize');
}

export function WindowSetMinSize(width, height) {
    if (isDesktop()) return window.runtime.WindowSetMinSize(width, height);
    warn('WindowSetMinSize');
}

export function WindowSetPosition(x, y) {
    if (isDesktop()) return window.runtime.WindowSetPosition(x, y);
    warn('WindowSetPosition');
}

export function WindowGetPosition() {
    if (isDesktop()) return window.runtime.WindowGetPosition();
    return { x: 0, y: 0 };
}

export function WindowHide() {
    if (isDesktop()) return window.runtime.WindowHide();
    warn('WindowHide');
}

export function WindowShow() {
    if (isDesktop()) return window.runtime.WindowShow();
    warn('WindowShow');
}

export function WindowMaximise() {
    if (isDesktop()) return window.runtime.WindowMaximise();
    warn('WindowMaximise');
}

export function WindowToggleMaximise() {
    if (isDesktop()) return window.runtime.WindowToggleMaximise();
    warn('WindowToggleMaximise');
}

export function WindowUnmaximise() {
    if (isDesktop()) return window.runtime.WindowUnmaximise();
    warn('WindowUnmaximise');
}

export function WindowIsMaximised() {
    if (isDesktop()) return window.runtime.WindowIsMaximised();
    return false;
}

export function WindowMinimise() {
    if (isDesktop()) return window.runtime.WindowMinimise();
    warn('WindowMinimise');
}

export function WindowUnminimise() {
    if (isDesktop()) return window.runtime.WindowUnminimise();
    warn('WindowUnminimise');
}

export function WindowSetBackgroundColour(R, G, B, A) {
    if (isDesktop()) return window.runtime.WindowSetBackgroundColour(R, G, B, A);
    warn('WindowSetBackgroundColour');
}

export function ScreenGetAll() {
    if (isDesktop()) return window.runtime.ScreenGetAll();
    return [];
}

export function WindowIsMinimised() {
    if (isDesktop()) return window.runtime.WindowIsMinimised();
    return false;
}

export function WindowIsNormal() {
    if (isDesktop()) return window.runtime.WindowIsNormal();
    return true;
}

export function BrowserOpenURL(url) {
    if (isDesktop()) return window.runtime.BrowserOpenURL(url);
    window.open(url, '_blank');
}

export function Environment() {
    if (isDesktop()) return window.runtime.Environment();
    return { platform: 'web', buildType: 'production' };
}

export function Quit() {
    if (isDesktop()) return window.runtime.Quit();
    warn('Quit');
}

export function Hide() {
    if (isDesktop()) return window.runtime.Hide();
    warn('Hide');
}

export function Show() {
    if (isDesktop()) return window.runtime.Show();
    warn('Show');
}

export function ClipboardGetText() {
    if (isDesktop()) return window.runtime.ClipboardGetText();
    warn('ClipboardGetText - use navigator.clipboard instead');
    return Promise.resolve('');
}

export function ClipboardSetText(text) {
    if (isDesktop()) return window.runtime.ClipboardSetText(text);
    warn('ClipboardSetText - use navigator.clipboard instead');
    return Promise.resolve();
}

export function OnFileDrop(callback, useDropTarget) {
    if (isDesktop()) return window.runtime.OnFileDrop(callback, useDropTarget);
    warn('OnFileDrop');
    return () => {};
}

export function OnFileDropOff() {
    if (isDesktop()) return window.runtime.OnFileDropOff();
    warn('OnFileDropOff');
}

export function CanResolveFilePaths() {
    if (isDesktop()) return window.runtime.CanResolveFilePaths();
    return false;
}

export function ResolveFilePaths(files) {
    if (isDesktop()) return window.runtime.ResolveFilePaths(files);
    return Promise.resolve([]);
}
