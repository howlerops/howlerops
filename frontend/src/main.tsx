import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { applyWailsClipboardFix } from './lib/wails-clipboard-fix'

// Apply WAILS-specific clipboard fixes before any components load
applyWailsClipboardFix()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
