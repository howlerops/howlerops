/**
 * Signup Form Component
 *
 * Handles user registration with validation.
 * Includes password strength indicators and confirmation.
 */

import { Check, Loader2, X } from 'lucide-react'
import { useState } from 'react'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/store/auth-store'

interface SignupFormProps {
  onSuccess?: () => void
}

export function SignupForm({ onSuccess }: SignupFormProps) {
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const { signUp, isLoading, error, clearError } = useAuthStore()

  const passwordsMatch = password === confirmPassword
  const passwordLengthOk = password.length >= 8
  const passwordHasUppercase = /[A-Z]/.test(password)
  const passwordHasNumber = /[0-9]/.test(password)
  const passwordStrong = passwordLengthOk && passwordHasUppercase && passwordHasNumber

  const canSubmit = username && email && password && passwordsMatch && passwordStrong

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    clearError()

    if (!passwordsMatch) {
      return
    }

    if (!passwordStrong) {
      return
    }

    try {
      await signUp(username, email, password)
      onSuccess?.()
    } catch (error) {
      // Error is already set in store
      console.error('Signup error:', error)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4 py-4">
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="space-y-2">
        <Label htmlFor="username">Username</Label>
        <Input
          id="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
          autoComplete="username"
          disabled={isLoading}
          placeholder="Choose a username"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="email">Email</Label>
        <Input
          id="email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          autoComplete="email"
          disabled={isLoading}
          placeholder="your.email@example.com"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="password">Password</Label>
        <Input
          id="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          autoComplete="new-password"
          disabled={isLoading}
          placeholder="Choose a strong password"
        />
        {password && (
          <div className="text-xs space-y-1 mt-2">
            <PasswordRequirement
              met={passwordLengthOk}
              text="At least 8 characters"
            />
            <PasswordRequirement
              met={passwordHasUppercase}
              text="One uppercase letter"
            />
            <PasswordRequirement met={passwordHasNumber} text="One number" />
          </div>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="confirmPassword">Confirm Password</Label>
        <Input
          id="confirmPassword"
          type="password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          required
          autoComplete="new-password"
          disabled={isLoading}
          placeholder="Confirm your password"
        />
        {confirmPassword && !passwordsMatch && (
          <p className="text-xs text-destructive">Passwords don't match</p>
        )}
      </div>

      <Button type="submit" className="w-full" disabled={isLoading || !canSubmit}>
        {isLoading ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Creating account...
          </>
        ) : (
          'Sign Up'
        )}
      </Button>
    </form>
  )
}

interface PasswordRequirementProps {
  met: boolean
  text: string
}

function PasswordRequirement({ met, text }: PasswordRequirementProps) {
  return (
    <div className="flex items-center gap-2">
      {met ? (
        <Check className="h-3 w-3 text-green-500 flex-shrink-0" />
      ) : (
        <X className="h-3 w-3 text-muted-foreground flex-shrink-0" />
      )}
      <span className={cn(met ? 'text-green-500' : 'text-muted-foreground')}>
        {text}
      </span>
    </div>
  )
}
