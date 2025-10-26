/**
 * Mock Wails runtime for web deployment
 * These stubs allow the code to build in Vercel while maintaining desktop app compatibility
 */

export function EventsOn(eventName: string, callback: (...args: any[]) => void): void {
  console.warn(`Desktop-only feature: EventsOn('${eventName}') is not available in web version`);
}

export function EventsOff(eventName: string): void {
  console.warn(`Desktop-only feature: EventsOff('${eventName}') is not available in web version`);
}

export function EventsEmit(eventName: string, ...args: any[]): void {
  console.warn(`Desktop-only feature: EventsEmit('${eventName}') is not available in web version`);
}

export function LogPrint(message: string): void {
  console.log(message);
}

export function LogDebug(message: string): void {
  console.debug(message);
}

export function LogInfo(message: string): void {
  console.info(message);
}

export function LogWarning(message: string): void {
  console.warn(message);
}

export function LogError(message: string): void {
  console.error(message);
}
