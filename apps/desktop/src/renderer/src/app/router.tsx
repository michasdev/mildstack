import { createHashRouter } from 'react-router'
import HomePage from '@renderer/app/routes/home'
import NotFoundPage from '@renderer/app/routes/not-found'

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
