/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useState } from 'react'
import { useParams } from 'react-router'

import { Badge } from '@renderer/components/ui/badge'
import { Button } from '@renderer/components/ui/button'
import { Separator } from '@renderer/components/ui/separator'
import { ObjectList } from './object-list'

export function BucketDetails() {
  const { bucketName } = useParams()
  const [activeTab, setActiveTab] = useState('objects')

  const tabs = [
    { id: 'objects', label: 'Objects' },
    { id: 'permissions', label: 'Permissions', disabled: true },
    { id: 'cors', label: 'CORS', disabled: true },
    { id: 'metrics', label: 'Metrics', disabled: true }
  ]

  return (
    <div className="flex h-full flex-col rounded-2xl border border-border bg-card shadow-xs/5">
      <div className="flex flex-col gap-3 border-b border-border px-4 py-3 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold">{bucketName}</h2>
            <Badge variant="outline" size="sm">
              Bucket details
            </Badge>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            Browse objects and prepare for future bucket-level settings.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {tabs.map((tab) => (
            <Button
              key={tab.id}
              variant={activeTab === tab.id ? 'default' : 'ghost'}
              size="sm"
              disabled={tab.disabled}
              onClick={() => !tab.disabled && setActiveTab(tab.id)}
            >
              {tab.label}
              {tab.disabled && (
                <span className="ml-2 rounded-full border border-border px-2 py-0.5 text-[10px] uppercase tracking-wider text-muted-foreground">
                  Soon
                </span>
              )}
            </Button>
          ))}
        </div>
      </div>

      <Separator />

      <div className="flex-1 min-h-0">
        {activeTab === 'objects' ? (
          <ObjectList />
        ) : (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            This section is coming soon.
          </div>
        )}
      </div>
    </div>
  )
}
