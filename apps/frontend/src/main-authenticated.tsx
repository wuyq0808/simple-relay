import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import AuthenticatedApp from './AuthenticatedApp.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AuthenticatedApp />
  </StrictMode>,
)