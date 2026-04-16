import { createHashRouter } from 'react-router'
import NotFoundPage from '@renderer/app/pages/not-found'
import ResourcesPage from '@renderer/features/resources/resources'

export const router = createHashRouter([
  {
    path: '/',
    element: <ResourcesPage />,
    children: [
      { index: true, element: <ResourcesPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
