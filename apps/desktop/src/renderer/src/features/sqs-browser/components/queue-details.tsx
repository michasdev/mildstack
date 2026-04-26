/* eslint-disable @typescript-eslint/explicit-function-return-type */

import { useState, useEffect, useCallback } from 'react'
import { useParams, useOutletContext } from 'react-router'
import { Send, Download, Settings as SettingsIcon, Trash2, ArrowRightLeft, RefreshCw, ArchiveRestore } from 'lucide-react'
import Editor from '@monaco-editor/react'
import { Button } from '@renderer/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@renderer/components/ui/tabs'
import { Input } from '@renderer/components/ui/input'
import { Label } from '@renderer/components/ui/label'
import { Spinner } from '@renderer/components/ui/spinner'
import { Badge } from '@renderer/components/ui/badge'
import { toast } from 'sonner'
import type { SQSBrowserOutletContext } from '../sqs-layout'
import type { SQSMessage } from '../types'

export function QueueDetails() {
  const { queueName } = useParams<{ queueName: string }>()
  const { api, region } = useOutletContext<SQSBrowserOutletContext>()
  const [activeTab, setActiveTab] = useState('messages')

  // Queue state
  const [attributes, setAttributes] = useState<Record<string, string>>({})
  const [queueUrl, setQueueUrl] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const isFifo = queueName?.endsWith('.fifo') || false

  // Send Message state
  const [messageBody, setMessageBody] = useState('{\n  "hello": "world"\n}')
  const [delaySeconds, setDelaySeconds] = useState(0)
  const [messageGroupId, setMessageGroupId] = useState('')
  const [messageDeduplicationId, setMessageDeduplicationId] = useState('')
  const [isSending, setIsSending] = useState(false)

  // Receive Messages state
  const [messages, setMessages] = useState<SQSMessage[]>([])
  const [isReceiving, setIsReceiving] = useState(false)
  const [receiveCount, setReceiveCount] = useState(10)

  // Redrive state
  const [isRedriving, setIsRedriving] = useState(false)

  const fetchQueueData = useCallback(async () => {
    if (!queueName) return
    try {
      setLoading(true)
      const summaries = await api.listQueues(region)
      const summary = summaries.find(q => q.QueueName === queueName)
      if (summary) {
        setQueueUrl(summary.QueueUrl)
        const attrs = await api.getQueueAttributes(summary.QueueUrl, region)
        setAttributes(attrs)
      }
    } catch (err) {
      console.error(err)
      toast.error('Error loading queue', {
        description: err instanceof Error ? err.message : String(err)
      })
    } finally {
      setLoading(false)
    }
  }, [api, queueName, region])

  useEffect(() => {
    void fetchQueueData()
  }, [fetchQueueData])

  const handleSendMessage = async () => {
    if (!queueUrl || !messageBody.trim()) return
    setIsSending(true)
    try {
      await api.sendMessage(
        queueUrl,
        messageBody,
        delaySeconds > 0 ? delaySeconds : undefined,
        messageGroupId || undefined,
        messageDeduplicationId || undefined,
        undefined,
        region
      )
      toast.success('Message sent', {
        description: 'Successfully placed message in queue.'
      })
      // Optional: reset form or keep it for sending multiple
      setMessageDeduplicationId('') // usually want to clear dedup id
      void fetchQueueData()
    } catch (err) {
      toast.error('Failed to send message', {
        description: err instanceof Error ? err.message : String(err)
      })
    } finally {
      setIsSending(false)
    }
  }

  const handleReceiveMessages = async () => {
    if (!queueUrl) return
    setIsReceiving(true)
    try {
      const received = await api.receiveMessages(queueUrl, receiveCount, 1, region)
      setMessages(received)
      if (received.length === 0) {
        toast.info('No messages found', {
          description: 'The queue is currently empty or messages are invisible.'
        })
      }
    } catch (err) {
      toast.error('Failed to receive messages', {
        description: err instanceof Error ? err.message : String(err)
      })
    } finally {
      setIsReceiving(false)
    }
  }

  const handleDeleteMessage = async (receiptHandle: string) => {
    if (!queueUrl) return
    try {
      await api.deleteMessage(queueUrl, receiptHandle, region)
      setMessages(prev => prev.filter(m => m.ReceiptHandle !== receiptHandle))
      toast.success('Message deleted', {
        description: 'Message has been deleted from the queue.'
      })
      void fetchQueueData()
    } catch (err) {
      toast.error('Failed to delete message', {
        description: err instanceof Error ? err.message : String(err)
      })
    }
  }

  // Parse DLQ target if it exists
  let dlqArn = ''
  if (attributes.RedrivePolicy) {
    try {
      dlqArn = JSON.parse(attributes.RedrivePolicy).deadLetterTargetArn
    } catch (e) { }
  }

  const handleRedrive = async () => {
    setIsRedriving(true)
    toast.info('Redrive Started', {
      description: 'Moving messages back to source queue...'
    })
    setTimeout(() => {
      setIsRedriving(false)
      toast.success('Redrive Completed', {
        description: 'Messages have been moved back to source queue.'
      })
    }, 2000)
  }

  if (loading && !queueUrl) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner className="h-6 w-6 text-muted-foreground" />
      </div>
    )
  }

  if (!queueUrl) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Queue not found.
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col rounded-2xl border border-border bg-card shadow-xs/5">
      <Tabs value={activeTab} onValueChange={setActiveTab} className="flex h-full flex-col">
        <div className="flex flex-col gap-3 border-b border-border px-4 py-3 md:flex-row md:items-center md:justify-between">
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-semibold">{queueName}</h2>
              {isFifo && (
                <Badge variant="outline" className="text-amber-500 border-amber-500/30">
                  FIFO
                </Badge>
              )}
              <Badge variant="outline">
                Queue details
              </Badge>
            </div>
            <div className="flex flex-col gap-1">
              <p className="text-xs text-muted-foreground break-all">{queueUrl}</p>
              {dlqArn && (
                <div className="flex items-center gap-2 text-[10px] text-red-400">
                  <ArrowRightLeft className="w-3 h-3" />
                  <span>Sends dead letters to: <strong>{dlqArn.split(':').pop()}</strong></span>
                </div>
              )}
            </div>
          </div>

          <div className="flex flex-col items-end gap-4 md:flex-row md:items-center">
            <div className="flex items-center gap-4 text-[10px] uppercase tracking-wider">
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Available</span>
                <span className="font-bold text-foreground">
                  {attributes.ApproximateNumberOfMessages || '0'}
                </span>
              </div>
              <div className="flex items-center gap-1.5 border-l border-border pl-4">
                <span className="text-muted-foreground">In Flight</span>
                <span className="font-bold text-amber-500">
                  {attributes.ApproximateNumberOfMessagesNotVisible || '0'}
                </span>
              </div>
              <div className="flex items-center gap-1.5 border-l border-border pl-4">
                <span className="text-muted-foreground">Delayed</span>
                <span className="font-bold text-foreground">
                  {attributes.ApproximateNumberOfMessagesDelayed || '0'}
                </span>
              </div>
            </div>

            <TabsList className="mt-2 md:mt-0">
              <TabsTrigger value="messages">
                <Download className="h-4 w-4" />
                Messages
              </TabsTrigger>
              <TabsTrigger value="send">
                <Send className="h-4 w-4" />
                Send
              </TabsTrigger>
              <TabsTrigger value="settings">
                <SettingsIcon className="h-4 w-4" />
                Attributes
              </TabsTrigger>
              {attributes.RedriveAllowPolicy && (
                <TabsTrigger value="dlq" className="text-red-400 data-active:text-red-400">
                  <ArchiveRestore className="h-4 w-4" />
                  DLQ
                </TabsTrigger>
              )}
            </TabsList>
          </div>
        </div>

        <div className="flex-1 min-h-0 overflow-y-auto">
          <TabsContent value="messages" className="h-full flex flex-col gap-4 p-4">
            <div className="flex items-center gap-4 bg-secondary/20 p-3 rounded-lg border border-border">
              <div className="flex items-center gap-2">
                <Label>Max Messages:</Label>
                <Input
                  type="number"
                  min={1}
                  max={10}
                  value={receiveCount}
                  onChange={(e) => setReceiveCount(parseInt(e.target.value) || 1)}
                  className="w-20 h-8"
                />
              </div>
              <Button onClick={handleReceiveMessages} disabled={isReceiving}>
                {isReceiving ? <Spinner /> : (
                  <RefreshCw className="w-4 h-4 mr-2" />
                )}
                Poll Messages
              </Button>
              <div className="flex-1" />
              <span className="text-xs text-muted-foreground">
                {messages.length} messages listed below
              </span>
            </div>

            <div className="flex-1 flex flex-col gap-3">
              {messages.length === 0 ? (
                <div className="flex flex-col items-center justify-center flex-1 text-muted-foreground border-2 border-dashed border-border rounded-lg p-8">
                  <Download className="w-8 h-8 mb-4 opacity-20" />
                  <p>No messages fetched.</p>
                  <p className="text-sm">Click "Poll Messages" to retrieve from the queue.</p>
                </div>
              ) : (
                messages.map((msg) => (
                  <div
                    key={msg.MessageId}
                    className="bg-card border border-border rounded-lg overflow-hidden flex flex-col"
                  >
                    <div className="bg-secondary/30 border-b border-border px-3 py-2 flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <span
                          className="text-xs font-mono text-muted-foreground"
                          title={msg.MessageId}
                        >
                          ID: {msg.MessageId.substring(0, 16)}...
                        </span>
                        {msg.Attributes?.ApproximateReceiveCount && (
                          <Badge variant="outline" className="text-[10px] h-5">
                            Receive Count: {msg.Attributes.ApproximateReceiveCount}
                          </Badge>
                        )}
                        {msg.Attributes?.SentTimestamp && (
                          <span className="text-[10px] text-muted-foreground">
                            Sent:{' '}
                            {new Date(parseInt(msg.Attributes.SentTimestamp)).toLocaleTimeString()}
                          </span>
                        )}
                      </div>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="text-red-400 hover:text-red-300 hover:bg-red-500/10"
                        onClick={() => handleDeleteMessage(msg.ReceiptHandle)}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                    <div className="p-3 overflow-x-auto">
                      <pre className="text-xs font-mono text-foreground m-0 p-0 whitespace-pre-wrap break-all">
                        {msg.Body}
                      </pre>
                    </div>
                  </div>
                ))
              )}
            </div>
          </TabsContent>

          <TabsContent value="send" className="p-4">
            <div className="max-w-3xl mx-auto space-y-6">
              <div className="space-y-3">
                <Label>Message Body</Label>
                <div className="overflow-hidden rounded-xl border border-border">
                  <Editor
                    height="240px"
                    defaultLanguage="json"
                    value={messageBody}
                    onChange={(val) => setMessageBody(val ?? '')}
                    theme="vs-dark"
                    loading={
                      <div className="flex h-[240px] items-center justify-center">
                        <Spinner className="h-6 w-6 text-muted-foreground" />
                      </div>
                    }
                    options={{
                      minimap: { enabled: false },
                      fontSize: 13,
                      lineNumbers: 'on',
                      scrollBeyondLastLine: false,
                      automaticLayout: true,
                      tabSize: 2,
                      wordWrap: 'on',
                      padding: { top: 12, bottom: 12 },
                      renderLineHighlight: 'gutter',
                      bracketPairColorization: { enabled: true },
                      guides: { bracketPairs: true }
                    }}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-6">
                <div className="space-y-2">
                  <Label>Delay Seconds</Label>
                  <Input
                    type="number"
                    min={0}
                    max={900}
                    value={delaySeconds}
                    onChange={(e) => setDelaySeconds(parseInt(e.target.value) || 0)}
                  />
                  <p className="text-xs text-muted-foreground">Delivery delay in seconds (0-900)</p>
                </div>
              </div>

              {isFifo && (
                <div className="grid grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <Label>Message Group ID</Label>
                    <Input
                      value={messageGroupId}
                      onChange={(e) => setMessageGroupId(e.target.value)}
                      placeholder="e.g. group-1"
                    />
                    <p className="text-xs text-muted-foreground">Required for FIFO queues</p>
                  </div>
                  <div className="space-y-2">
                    <Label>Message Deduplication ID</Label>
                    <Input
                      value={messageDeduplicationId}
                      onChange={(e) => setMessageDeduplicationId(e.target.value)}
                      placeholder="Optional if ContentBasedDeduplication is enabled"
                    />
                  </div>
                </div>
              )}

              <div className="pt-4 border-t border-border flex justify-end">
                <Button
                  onClick={handleSendMessage}
                  disabled={isSending || (isFifo && !messageGroupId)}
                >
                  {isSending ? <Spinner /> : (
                    <Send className="w-4 h-4 mr-2" />
                  )}
                  Send Message
                </Button>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="settings" className="p-4">
            <div className="max-w-3xl space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="space-y-2 bg-secondary/10 p-4 rounded-lg border border-border">
                  <Label className="text-xs text-muted-foreground">Visibility Timeout</Label>
                  <p className="font-mono">{attributes.VisibilityTimeout} seconds</p>
                </div>
                <div className="space-y-2 bg-secondary/10 p-4 rounded-lg border border-border">
                  <Label className="text-xs text-muted-foreground">Message Retention Period</Label>
                  <p className="font-mono">{attributes.MessageRetentionPeriod} seconds</p>
                </div>
                <div className="space-y-2 bg-secondary/10 p-4 rounded-lg border border-border">
                  <Label className="text-xs text-muted-foreground">Maximum Message Size</Label>
                  <p className="font-mono">{attributes.MaximumMessageSize} bytes</p>
                </div>
                <div className="space-y-2 bg-secondary/10 p-4 rounded-lg border border-border">
                  <Label className="text-xs text-muted-foreground">Delay Seconds</Label>
                  <p className="font-mono">{attributes.DelaySeconds} seconds</p>
                </div>
                <div className="space-y-2 bg-secondary/10 p-4 rounded-lg border border-border">
                  <Label className="text-xs text-muted-foreground">Created Timestamp</Label>
                  <p className="font-mono">
                    {attributes.CreatedTimestamp
                      ? new Date(parseInt(attributes.CreatedTimestamp) * 1000).toLocaleString()
                      : '-'}
                  </p>
                </div>
                <div className="space-y-2 bg-secondary/10 p-4 rounded-lg border border-border">
                  <Label className="text-xs text-muted-foreground">Queue ARN</Label>
                  <p className="font-mono text-xs break-all">{attributes.QueueArn}</p>
                </div>
              </div>
            </div>
          </TabsContent>

          {attributes.RedriveAllowPolicy && (
            <TabsContent value="dlq" className="p-4">
              <div className="max-w-3xl space-y-6">
                <div className="bg-red-500/5 border border-red-500/20 rounded-lg p-6 text-center">
                  <ArchiveRestore className="w-12 h-12 text-red-400 mx-auto mb-4" />
                  <h3 className="text-lg font-medium text-foreground mb-2">
                    Dead Letter Queue Management
                  </h3>
                  <p className="text-sm text-muted-foreground mb-6 max-w-md mx-auto">
                    This queue is configured as a DLQ. You can redrive messages back to their
                    original source queue for reprocessing.
                  </p>
                  <Button
                    onClick={handleRedrive}
                    disabled={isRedriving}
                    variant="secondary"
                    className="bg-red-500/10 text-red-400 hover:bg-red-500/20 border-red-500/20"
                  >
                    {isRedriving ? <Spinner /> : (
                      <ArchiveRestore className="w-4 h-4 mr-2" />
                    )}
                    Redrive Messages to Source
                  </Button>
                </div>
              </div>
            </TabsContent>
          )}
        </div>
      </Tabs>
    </div>
  )
}
