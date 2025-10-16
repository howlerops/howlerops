import { createRoot } from 'react-dom/client'
import './index.css'
import App from './app.tsx'
import { applyWailsClipboardFix } from './lib/wails-clipboard-fix'

// Apply WAILS-specific clipboard fixes before any components load
applyWailsClipboardFix()

// Note: StrictMode disabled in development to prevent double-mounting
// which causes Wails callback registration issues during HMR
createRoot(document.getElementById('root')!).render(
  <App />
)
