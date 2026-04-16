import { createHashRouter } from 'react-router'
import HomePage from '@renderer/app/pages/home'
import NotFoundPage from '@renderer/app/pages/not-found'

export const router = createHashRouter([
  {
    path: '/',
    element: <HomePage />,
    children: [
      { index: true, element: <HomePage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
