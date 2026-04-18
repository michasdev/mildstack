import { createHashRouter } from 'react-router'
import NotFoundPage from '@renderer/shared/not-found'
import ResourcesPage from '@renderer/features/resources/resources'
import InstancesPage from '@renderer/features/instances/instances'
import { Layout } from '@renderer/shared/layout'
import { S3Layout } from '@renderer/features/s3-browser/s3-layout'
import { BucketsList } from '@renderer/features/s3-browser/components/buckets-list'
import { BucketDetails } from '@renderer/features/s3-browser/components/bucket-details'

export const router = createHashRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <InstancesPage /> },
      { path: '/resources', element: <ResourcesPage /> },
      { 
        path: '/resources/s3', 
        element: <S3Layout />,
        children: [
          { index: true, element: <BucketsList /> },
          { path: ':bucketName/*', element: <BucketDetails /> }
        ]
      },
      { path: '/instances/:instanceid/resources', element: <ResourcesPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
