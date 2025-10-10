import { NavigateFunction } from 'react-router-dom'

class NavigationService {
  private navigate: NavigateFunction | null = null

  setNavigate(navigate: NavigateFunction) {
    this.navigate = navigate
  }

  to(path: string) {
    if (this.navigate) {
      this.navigate(path)
    } else {
      console.error('Navigation not initialized')
    }
  }
}

export const navigationService = new NavigationService()