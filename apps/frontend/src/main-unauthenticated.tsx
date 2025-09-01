import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import UnauthenticatedApp from './UnauthenticatedApp.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <UnauthenticatedApp />
  </StrictMode>,
)