import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { navigationService } from '@/lib/navigation'

export function NavigationProvider({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()

  useEffect(() => {
    navigationService.setNavigate(navigate)
  }, [navigate])

  return <>{children}</>
}