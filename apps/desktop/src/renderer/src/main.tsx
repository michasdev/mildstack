import './assets/main.css'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import { router } from '@renderer/router'
import { AnchoredToastProvider, ToastProvider } from '@renderer/components/ui/toast'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ToastProvider>
      <AnchoredToastProvider>
        <RouterProvider router={router} />
      </AnchoredToastProvider>
    </ToastProvider>
  </StrictMode>
)
