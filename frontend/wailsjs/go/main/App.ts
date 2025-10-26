/**
 * Mock Wails bindings for web deployment
 * These stubs allow the code to build in Vercel while maintaining desktop app compatibility
 */

export function FixSQLErrorWithOptions(): Promise<any> {
  throw new Error('Desktop-only feature: FixSQLErrorWithOptions is not available in web version');
}

export function GenerateSQL(): Promise<any> {
  throw new Error('Desktop-only feature: GenerateSQL is not available in web version');
}

export function ExplainQuery(): Promise<any> {
  throw new Error('Desktop-only feature: ExplainQuery is not available in web version');
}

export function OptimizeQuery(): Promise<any> {
  throw new Error('Desktop-only feature: OptimizeQuery is not available in web version');
}
