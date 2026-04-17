import { createHashRouter } from 'react-router'
import NotFoundPage from '@renderer/shared/not-found'
import ResourcesPage from '@renderer/features/resources/resources'
import InstancesPage from '@renderer/features/instances/instances'
import { Layout } from '@renderer/shared/layout'

export const router = createHashRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <InstancesPage /> },
      { path: '/resources', element: <ResourcesPage /> },
      { path: '/instances/:instanceid/resources', element: <ResourcesPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
