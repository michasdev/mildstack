import './assets/main.css'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router'
import { router } from '@renderer/router'
import { AnchoredToastProvider, ToastProvider } from '@renderer/components/ui/toast'
import { TooltipProvider } from '@renderer/components/ui/tooltip'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ToastProvider>
      <AnchoredToastProvider>
        <TooltipProvider>
          <RouterProvider router={router} />
        </TooltipProvider>
      </AnchoredToastProvider>
    </ToastProvider>
  </StrictMode>
)
