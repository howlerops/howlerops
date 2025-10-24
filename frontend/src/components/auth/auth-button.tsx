/**
 * Auth Button Component
 *
 * Displays sign in button for unauthenticated users,
 * or user menu dropdown for authenticated users.
 */

import { useState } from 'react'
import { useAuthStore } from '@/store/auth-store'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { User, LogOut, Settings, Shield } from 'lucide-react'
import { AuthModal } from './auth-modal'
import { useNavigate } from 'react-router-dom'

export function AuthButton() {
  const { isAuthenticated, user, signOut } = useAuthStore()
  const [showAuthModal, setShowAuthModal] = useState(false)
  const navigate = useNavigate()

  const handleSignOut = async () => {
    try {
      await signOut()
    } catch (error) {
      console.error('Sign out error:', error)
    }
  }

  if (!isAuthenticated) {
    return (
      <>
        <Button
          onClick={() => setShowAuthModal(true)}
          variant="outline"
          size="sm"
        >
          Sign In
        </Button>
        <AuthModal open={showAuthModal} onOpenChange={setShowAuthModal} />
      </>
    )
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm">
            <User className="h-4 w-4 mr-2" />
            {user?.username}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>
            <div className="flex flex-col space-y-1">
              <p className="text-sm font-medium leading-none">{user?.username}</p>
              <p className="text-xs leading-none text-muted-foreground">
                {user?.email}
              </p>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => navigate('/settings')}>
            <Settings className="h-4 w-4 mr-2" />
            Settings
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate('/settings?tab=tier')}>
            <Shield className="h-4 w-4 mr-2" />
            Subscription
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleSignOut}>
            <LogOut className="h-4 w-4 mr-2" />
            Sign Out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <AuthModal open={showAuthModal} onOpenChange={setShowAuthModal} />
    </>
  )
}
