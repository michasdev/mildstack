import { createHashRouter } from 'react-router'
import NotFoundPage from '@renderer/shared/not-found'
import InstancesPage from '@renderer/features/instances/instances'
import { Layout } from '@renderer/shared/layout'
import { S3Layout } from '@renderer/features/s3-browser/s3-layout'
import { BucketsList } from '@renderer/features/s3-browser/components/buckets-list'
import { BucketDetails } from '@renderer/features/s3-browser/components/bucket-details'
import { DynamoDBLayout } from '@renderer/features/dynamodb-browser/dynamodb-layout'
import { TablesList } from '@renderer/features/dynamodb-browser/components/tables-list'
import { TableDetails } from '@renderer/features/dynamodb-browser/components/table-details'
import { SQSLayout } from '@renderer/features/sqs-browser/sqs-layout'
import { QueuesList } from '@renderer/features/sqs-browser/components/queues-list'
import { QueueDetails } from '@renderer/features/sqs-browser/components/queue-details'
import { SNSLayout } from '@renderer/features/sns-browser/sns-layout'
import { SNSBrowser } from '@renderer/features/sns-browser/sns-browser'
import SettingsPage from '@renderer/features/settings/settings'

export const router = createHashRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <InstancesPage /> },
      { 
        path: '/resources/s3', 
        element: <S3Layout />,
        children: [
          { index: true, element: <BucketsList /> },
          { path: ':bucketName/*', element: <BucketDetails /> }
        ]
      },
      {
        path: '/resources/dynamodb',
        element: <DynamoDBLayout />,
        children: [
          { index: true, element: <TablesList /> },
          { path: ':tableName/*', element: <TableDetails /> }
        ]
      },
      {
        path: '/resources/sqs',
        element: <SQSLayout />,
        children: [
          { index: true, element: <QueuesList /> },
          { path: ':queueName/*', element: <QueueDetails /> }
        ]
      },
      {
        path: '/resources/sns',
        element: <SNSLayout />,
        children: [
          { index: true, element: <SNSBrowser /> },
          { path: ':topicName/*', element: <SNSBrowser /> }
        ]
      },
      { path: '/settings', element: <SettingsPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])
