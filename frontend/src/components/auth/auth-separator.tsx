/**
 * Auth Separator Component
 *
 * Visual separator for authentication methods with "Or continue with" text.
 * Creates a horizontal line with centered text overlay.
 */

export function AuthSeparator() {
  return (
    <div className="relative">
      <div className="absolute inset-0 flex items-center">
        <span className="w-full border-t" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-card px-2 text-muted-foreground">
          Or continue with
        </span>
      </div>
    </div>
  )
}
