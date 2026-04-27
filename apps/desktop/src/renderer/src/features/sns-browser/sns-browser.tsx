/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useOutletContext, useParams, useSearchParams } from 'react-router'
import {
  Bell,
  Layers3,
  MessageSquareText,
  Plus,
  Search,
  Smartphone,
  Trash2,
  Users,
  Send,
  RefreshCw,
  ShieldCheck
} from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import { Input } from '@renderer/components/ui/input'
import { Textarea } from '@renderer/components/ui/textarea'
import { Badge } from '@renderer/components/ui/badge'
import { Spinner } from '@renderer/components/ui/spinner'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@renderer/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@renderer/components/ui/tabs'
import SpotlightCard from '@renderer/components/ui/spotlight-card'
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@renderer/components/ui/empty'
import { CardHeader, CardTitle, CardDescription, CardAction } from '@renderer/components/ui/card'
import { Separator } from '@renderer/components/ui/separator'
import { toast } from 'sonner'

import type {
  SNSBrowserOutletContext
} from './sns-layout'
import type {
  SNSTopicSummary,
  SNSSubscriptionSummary,
  SNSPlatformApplicationSummary,
  SNSPlatformEndpointSummary,
  SMSSandboxPhoneNumber,
  SNSBrowserSection
} from './types'
import {
  formatAttributesJson,
  parseAttributesJson,
  topicNameFromArn
} from './types'

function sectionLabel(section: SNSBrowserSection): string {
  switch (section) {
    case 'topics':
      return 'Topics'
    case 'subscriptions':
      return 'Subscriptions'
    case 'platform-applications':
      return 'Platform Applications'
    case 'sms':
      return 'SMS Sandbox'
  }
}

function splitCsv(input: string): string[] {
  return input
    .split(',')
    .map((value) => value.trim())
    .filter(Boolean)
}

export function SNSBrowser() {
  const { api, region } = useOutletContext<SNSBrowserOutletContext>()
  const navigate = useNavigate()
  const { topicName: topicNameParam } = useParams<{ topicName?: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const [activeSection, setActiveSection] = useState<SNSBrowserSection>('topics')
  const [loading, setLoading] = useState(true)

  const [topics, setTopics] = useState<SNSTopicSummary[]>([])
  const [subscriptions, setSubscriptions] = useState<SNSSubscriptionSummary[]>([])
  const [platformApps, setPlatformApps] = useState<SNSPlatformApplicationSummary[]>([])
  const [smsNumbers, setSmsNumbers] = useState<SMSSandboxPhoneNumber[]>([])
  const [optedOutNumbers, setOptedOutNumbers] = useState<string[]>([])
  const [originationNumbers, setOriginationNumbers] = useState<Array<{ PhoneNumber?: string; Status?: string; CreatedAt?: string }>>([])
  const [smsSandbox, setSmsSandbox] = useState<boolean>(false)
  const [smsAttributes, setSmsAttributes] = useState<Record<string, string>>({})

  const [search, setSearch] = useState('')

  const [selectedTopicArn, setSelectedTopicArn] = useState<string>('')
  const [topicAttributes, setTopicAttributes] = useState<Record<string, string>>({})
  const [topicSubscriptions, setTopicSubscriptions] = useState<SNSSubscriptionSummary[]>([])
  const [topicTags, setTopicTags] = useState<Array<{ Key?: string; Value?: string }>>([])
  const [topicPolicy, setTopicPolicy] = useState('')
  const [newTopicName, setNewTopicName] = useState('')
  const [newTopicAttributes, setNewTopicAttributes] = useState('')
  const [isCreateTopicOpen, setIsCreateTopicOpen] = useState(false)
  const [isCreatingTopic, setIsCreatingTopic] = useState(false)
  const [topicDetailsLoading, setTopicDetailsLoading] = useState(false)
  const [topicAttributeName, setTopicAttributeName] = useState('')
  const [topicAttributeValue, setTopicAttributeValue] = useState('')
  const [topicMessage, setTopicMessage] = useState('Hello from MildStack SNS')
  const [topicSubject, setTopicSubject] = useState('MildStack')
  const [topicPublishStatus, setTopicPublishStatus] = useState('')
  const [topicBatchMessageOne, setTopicBatchMessageOne] = useState('Hello from batch 1')
  const [topicBatchMessageTwo, setTopicBatchMessageTwo] = useState('Hello from batch 2')
  const [topicTagKey, setTopicTagKey] = useState('')
  const [topicTagValue, setTopicTagValue] = useState('')
  const [topicPermissionLabel, setTopicPermissionLabel] = useState('')
  const [topicPermissionAccounts, setTopicPermissionAccounts] = useState('')
  const [topicPermissionActions, setTopicPermissionActions] = useState('Publish,Subscribe')

  const [selectedSubscriptionArn, setSelectedSubscriptionArn] = useState<string>('')
  const [subscriptionAttributes, setSubscriptionAttributes] = useState<Record<string, string>>({})
  const [subscriptionTopicArn, setSubscriptionTopicArn] = useState('')
  const [subscriptionProtocol, setSubscriptionProtocol] = useState('email')
  const [subscriptionEndpoint, setSubscriptionEndpoint] = useState('')
  const [subscriptionToken, setSubscriptionToken] = useState('')
  const [subscriptionAttributeName, setSubscriptionAttributeName] = useState('')
  const [subscriptionAttributeValue, setSubscriptionAttributeValue] = useState('')

  const [selectedAppArn, setSelectedAppArn] = useState<string>('')
  const [platformAppAttributes, setPlatformAppAttributes] = useState<Record<string, string>>({})
  const [platformEndpoints, setPlatformEndpoints] = useState<SNSPlatformEndpointSummary[]>([])
  const [newPlatformAppName, setNewPlatformAppName] = useState('')
  const [newPlatformAppPlatform, setNewPlatformAppPlatform] = useState('GCM')
  const [newPlatformAppAttributes, setNewPlatformAppAttributes] = useState('{}')
  const [newEndpointToken, setNewEndpointToken] = useState('')
  const [newEndpointCustomUserData, setNewEndpointCustomUserData] = useState('')
  const [newEndpointAttributes, setNewEndpointAttributes] = useState('{}')
  const [platformAttributeName, setPlatformAttributeName] = useState('')
  const [platformAttributeValue, setPlatformAttributeValue] = useState('')

  const [newSmsPhoneNumber, setNewSmsPhoneNumber] = useState('')
  const [newSmsLanguageCode, setNewSmsLanguageCode] = useState('en-US')
  const [verifySmsPhoneNumber, setVerifySmsPhoneNumber] = useState('')
  const [verifySmsOtp, setVerifySmsOtp] = useState('')
  const [optOutPhoneNumber, setOptOutPhoneNumber] = useState('')
  const [optInPhoneNumber, setOptInPhoneNumber] = useState('')
  const [checkOptedOutPhoneNumber, setCheckOptedOutPhoneNumber] = useState('')
  const [smsAttributeName, setSmsAttributeName] = useState('')
  const [smsAttributeValue, setSmsAttributeValue] = useState('')
  const currentTopicName = topicNameParam ? decodeURIComponent(topicNameParam) : ''
  const currentTopicArn = useMemo(() => {
    if (!currentTopicName) return ''
    return topics.find(
      (topic) => topic.TopicName === currentTopicName || topicNameFromArn(topic.TopicArn) === currentTopicName
    )?.TopicArn ?? ''
  }, [currentTopicName, topics])

  useEffect(() => {
    const section = searchParams.get('section')
    if (section === 'subscriptions' || section === 'platform-applications' || section === 'sms') {
      setActiveSection(section)
      return
    }
    setActiveSection('topics')
  }, [searchParams])

  const fetchTopics = useCallback(async () => {
    const response = await api.listTopics(region)
    setTopics(response)
  }, [api, region])

  const fetchSubscriptions = useCallback(async () => {
    const response = await api.listSubscriptions(region)
    setSubscriptions(response)
  }, [api, region])

  const fetchPlatformApps = useCallback(async () => {
    const response = await api.listPlatformApplications(region)
    setPlatformApps(response)
  }, [api, region])

  const fetchSmsData = useCallback(async () => {
    const [sandboxStatus, attributes, numbers, optedOut, origination] = await Promise.all([
      api.getSMSSandboxAccountStatus(region),
      api.getSMSAttributes(undefined, region),
      api.listSMSSandboxPhoneNumbers(region),
      api.listPhoneNumbersOptedOut(region),
      api.listOriginationNumbers(region)
    ])

    setSmsSandbox(sandboxStatus)
    setSmsAttributes(attributes)
    setSmsNumbers(numbers)
    setOptedOutNumbers(optedOut)
    setOriginationNumbers(origination)
  }, [api, region])

  const fetchTopicDetails = useCallback(async (topicArn: string) => {
    if (!topicArn) return
    setTopicDetailsLoading(true)
    try {
      const [attributes, subscriptionsResponse, tags, policy] = await Promise.all([
        api.getTopicAttributes(topicArn, region),
        api.listSubscriptionsByTopic(topicArn, region),
        api.listTagsForResource(topicArn, region).catch(() => []),
        api.getDataProtectionPolicy(topicArn, region).catch(() => '')
      ])
      setTopicAttributes(attributes)
      setTopicSubscriptions(subscriptionsResponse)
      setTopicTags(tags)
      setTopicPolicy(policy)
    } finally {
      setTopicDetailsLoading(false)
    }
  }, [api, region])

  const fetchSubscriptionDetails = useCallback(async (subscriptionArn: string) => {
    if (!subscriptionArn) return
    const attributes = await api.getSubscriptionAttributes(subscriptionArn, region)
    setSubscriptionAttributes(attributes)
  }, [api, region])

  const fetchPlatformAppDetails = useCallback(async (platformApplicationArn: string) => {
    if (!platformApplicationArn) return
    const [attributes, endpoints] = await Promise.all([
      api.getPlatformApplicationAttributes(platformApplicationArn, region),
      api.listEndpointsByPlatformApplication(platformApplicationArn, region)
    ])
    setPlatformAppAttributes(attributes)
    setPlatformEndpoints(endpoints)
  }, [api, region])

  useEffect(() => {
    let cancelled = false

    const run = async () => {
      setLoading(true)
      try {
        if (activeSection === 'topics') {
          await fetchTopics()
        } else if (activeSection === 'subscriptions') {
          await fetchSubscriptions()
        } else if (activeSection === 'platform-applications') {
          await fetchPlatformApps()
        } else {
          await fetchSmsData()
        }
      } catch (error) {
        console.error(error)
        if (!cancelled) {
          toast.error(`Failed to load ${sectionLabel(activeSection).toLowerCase()}`, {
            description: error instanceof Error ? error.message : String(error)
          })
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void run()
    return () => {
      cancelled = true
    }
  }, [activeSection, fetchPlatformApps, fetchSmsData, fetchSubscriptions, fetchTopics])

  useEffect(() => {
    if (currentTopicArn) {
      void fetchTopicDetails(currentTopicArn)
    }
  }, [currentTopicArn, fetchTopicDetails])

  useEffect(() => {
    if (selectedSubscriptionArn) {
      void fetchSubscriptionDetails(selectedSubscriptionArn)
    }
  }, [fetchSubscriptionDetails, selectedSubscriptionArn])

  useEffect(() => {
    if (selectedAppArn) {
      void fetchPlatformAppDetails(selectedAppArn)
    }
  }, [fetchPlatformAppDetails, selectedAppArn])

  useEffect(() => {
    if (!currentTopicName) {
      setSelectedTopicArn('')
      return
    }

    setSelectedTopicArn(currentTopicArn)
  }, [currentTopicArn, currentTopicName])

  const filteredTopics = useMemo(
    () => topics.filter((topic) => topic.TopicName.toLowerCase().includes(search.toLowerCase()) || topic.TopicArn.toLowerCase().includes(search.toLowerCase())),
    [search, topics]
  )

  const filteredSubscriptions = useMemo(
    () => subscriptions.filter((subscription) =>
      subscription.TopicName.toLowerCase().includes(search.toLowerCase()) ||
      subscription.Endpoint.toLowerCase().includes(search.toLowerCase()) ||
      subscription.SubscriptionArn.toLowerCase().includes(search.toLowerCase())
    ),
    [search, subscriptions]
  )

  const filteredApps = useMemo(
    () => platformApps.filter((app) => app.Name.toLowerCase().includes(search.toLowerCase()) || app.PlatformApplicationArn.toLowerCase().includes(search.toLowerCase())),
    [platformApps, search]
  )

  const filteredSmsNumbers = useMemo(
    () => smsNumbers.filter((entry) => entry.PhoneNumber.toLowerCase().includes(search.toLowerCase())),
    [search, smsNumbers]
  )

  const resetCreateTopicForm = () => {
    setNewTopicName('')
    setNewTopicAttributes('')
  }

  const handleCreateTopic = async () => {
    const topicName = newTopicName.trim()
    if (!topicName) return

    setIsCreatingTopic(true)
    try {
      const attributes = newTopicAttributes.trim() ? parseAttributesJson(newTopicAttributes) : {}
      await api.createTopic(topicName, attributes, region)
      setIsCreateTopicOpen(false)
      resetCreateTopicForm()
      await fetchTopics()
      toast.success('Topic created')
    } catch (error) {
      toast.error('Failed to create topic', { description: error instanceof Error ? error.message : String(error) })
    } finally {
      setIsCreatingTopic(false)
    }
  }

  const handleDeleteTopic = async (topicArn: string) => {
    try {
      await api.deleteTopic(topicArn, region)
      if (selectedTopicArn === topicArn) setSelectedTopicArn('')
      await fetchTopics()
      toast.success('Topic deleted')
    } catch (error) {
      toast.error('Failed to delete topic', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleSetTopicAttribute = async () => {
    if (!selectedTopicArn || !topicAttributeName.trim()) return
    try {
      await api.setTopicAttribute(selectedTopicArn, topicAttributeName.trim(), topicAttributeValue, region)
      setTopicAttributeName('')
      setTopicAttributeValue('')
      await fetchTopicDetails(selectedTopicArn)
      toast.success('Topic attribute updated')
    } catch (error) {
      toast.error('Failed to update topic attribute', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handlePublishTopic = async () => {
    if (!selectedTopicArn || !topicMessage.trim()) return
    try {
      const messageId = await api.publish(
        selectedTopicArn,
        undefined,
        undefined,
        topicMessage,
        topicSubject || undefined,
        undefined,
        undefined,
        undefined,
        undefined,
        region
      )
      setTopicPublishStatus(`Message sent: ${messageId}`)
      toast.success('Message published')
    } catch (error) {
      toast.error('Failed to publish message', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handlePublishBatch = async () => {
    if (!selectedTopicArn) return
    try {
      await api.publishBatch(selectedTopicArn, [
        { Id: '1', Message: topicBatchMessageOne },
        { Id: '2', Message: topicBatchMessageTwo }
      ], region)
      toast.success('Batch published')
    } catch (error) {
      toast.error('Failed to publish batch', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleAddTopicTag = async () => {
    if (!selectedTopicArn || !topicTagKey.trim()) return
    try {
      await api.tagResource(selectedTopicArn, { [topicTagKey.trim()]: topicTagValue }, region)
      setTopicTagKey('')
      setTopicTagValue('')
      await fetchTopicDetails(selectedTopicArn)
      toast.success('Tag added')
    } catch (error) {
      toast.error('Failed to add tag', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleRemoveTopicTag = async (tagKey: string) => {
    if (!selectedTopicArn) return
    try {
      await api.untagResource(selectedTopicArn, [tagKey], region)
      await fetchTopicDetails(selectedTopicArn)
      toast.success('Tag removed')
    } catch (error) {
      toast.error('Failed to remove tag', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleAddPermission = async () => {
    if (!selectedTopicArn || !topicPermissionLabel.trim()) return
    try {
      await api.addPermission(
        selectedTopicArn,
        topicPermissionLabel.trim(),
        splitCsv(topicPermissionAccounts),
        splitCsv(topicPermissionActions),
        region
      )
      toast.success('Permission added')
    } catch (error) {
      toast.error('Failed to add permission', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleRemovePermission = async () => {
    if (!selectedTopicArn || !topicPermissionLabel.trim()) return
    try {
      await api.removePermission(selectedTopicArn, topicPermissionLabel.trim(), region)
      toast.success('Permission removed')
    } catch (error) {
      toast.error('Failed to remove permission', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleSavePolicy = async () => {
    if (!selectedTopicArn) return
    try {
      await api.putDataProtectionPolicy(selectedTopicArn, topicPolicy, region)
      toast.success('Policy saved')
    } catch (error) {
      toast.error('Failed to save policy', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleSubscribe = async () => {
    if (!subscriptionTopicArn.trim() || !subscriptionEndpoint.trim()) return
    try {
      await api.subscribe(
        subscriptionTopicArn.trim(),
        subscriptionProtocol.trim(),
        subscriptionEndpoint.trim(),
        undefined,
        true,
        region
      )
      setSubscriptionTopicArn('')
      setSubscriptionEndpoint('')
      await fetchSubscriptions()
      toast.success('Subscription created')
    } catch (error) {
      toast.error('Failed to create subscription', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleConfirmSubscription = async () => {
    if (!subscriptionTopicArn.trim() || !subscriptionToken.trim()) return
    try {
      await api.confirmSubscription(subscriptionTopicArn.trim(), subscriptionToken.trim(), region)
      setSubscriptionToken('')
      await fetchSubscriptions()
      toast.success('Subscription confirmed')
    } catch (error) {
      toast.error('Failed to confirm subscription', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleSetSubscriptionAttribute = async () => {
    if (!selectedSubscriptionArn || !subscriptionAttributeName.trim()) return
    try {
      await api.setSubscriptionAttribute(selectedSubscriptionArn, subscriptionAttributeName.trim(), subscriptionAttributeValue, region)
      setSubscriptionAttributeName('')
      setSubscriptionAttributeValue('')
      await fetchSubscriptionDetails(selectedSubscriptionArn)
      toast.success('Subscription attribute updated')
    } catch (error) {
      toast.error('Failed to update subscription attribute', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleDeleteSubscription = async (subscriptionArn: string) => {
    try {
      await api.unsubscribe(subscriptionArn, region)
      if (selectedSubscriptionArn === subscriptionArn) setSelectedSubscriptionArn('')
      await fetchSubscriptions()
      toast.success('Subscription deleted')
    } catch (error) {
      toast.error('Failed to delete subscription', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleCreatePlatformApp = async () => {
    try {
      const attrs = parseAttributesJson(newPlatformAppAttributes)
      await api.createPlatformApplication(newPlatformAppName.trim(), newPlatformAppPlatform.trim(), attrs, region)
      setNewPlatformAppName('')
      setNewPlatformAppAttributes('{}')
      await fetchPlatformApps()
      toast.success('Platform application created')
    } catch (error) {
      toast.error('Failed to create platform application', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleDeletePlatformApp = async (platformApplicationArn: string) => {
    try {
      await api.deletePlatformApplication(platformApplicationArn, region)
      if (selectedAppArn === platformApplicationArn) setSelectedAppArn('')
      await fetchPlatformApps()
      toast.success('Platform application deleted')
    } catch (error) {
      toast.error('Failed to delete platform application', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleSetPlatformAppAttribute = async () => {
    if (!selectedAppArn || !platformAttributeName.trim()) return
    try {
      await api.setPlatformApplicationAttributes(selectedAppArn, { [platformAttributeName.trim()]: platformAttributeValue }, region)
      setPlatformAttributeName('')
      setPlatformAttributeValue('')
      await fetchPlatformAppDetails(selectedAppArn)
      toast.success('Platform application attribute updated')
    } catch (error) {
      toast.error('Failed to update platform application attribute', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleCreateEndpoint = async () => {
    if (!selectedAppArn || !newEndpointToken.trim()) return
    try {
      const attrs = parseAttributesJson(newEndpointAttributes)
      await api.createPlatformEndpoint(
        selectedAppArn,
        newEndpointToken.trim(),
        newEndpointCustomUserData.trim() || undefined,
        attrs,
        region
      )
      setNewEndpointToken('')
      setNewEndpointCustomUserData('')
      setNewEndpointAttributes('{}')
      await fetchPlatformAppDetails(selectedAppArn)
      toast.success('Endpoint created')
    } catch (error) {
      toast.error('Failed to create endpoint', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleDeleteEndpoint = async (endpointArn: string) => {
    try {
      await api.deleteEndpoint(endpointArn, region)
      await fetchPlatformAppDetails(selectedAppArn)
      toast.success('Endpoint deleted')
    } catch (error) {
      toast.error('Failed to delete endpoint', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleSetSmsAttributes = async () => {
    if (!smsAttributeName.trim()) return
    try {
      await api.setSMSAttributes({ [smsAttributeName.trim()]: smsAttributeValue }, region)
      setSmsAttributeName('')
      setSmsAttributeValue('')
      await fetchSmsData()
      toast.success('SMS attributes updated')
    } catch (error) {
      toast.error('Failed to update SMS attributes', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleCreateSmsSandboxNumber = async () => {
    if (!newSmsPhoneNumber.trim()) return
    try {
      await api.createSMSSandboxPhoneNumber(newSmsPhoneNumber.trim(), newSmsLanguageCode.trim(), region)
      setNewSmsPhoneNumber('')
      await fetchSmsData()
      toast.success('SMS sandbox phone number added')
    } catch (error) {
      toast.error('Failed to add SMS sandbox phone number', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleVerifySmsSandboxNumber = async () => {
    if (!verifySmsPhoneNumber.trim() || !verifySmsOtp.trim()) return
    try {
      await api.verifySMSSandboxPhoneNumber(verifySmsPhoneNumber.trim(), verifySmsOtp.trim(), region)
      setVerifySmsPhoneNumber('')
      setVerifySmsOtp('')
      await fetchSmsData()
      toast.success('SMS sandbox phone number verified')
    } catch (error) {
      toast.error('Failed to verify SMS sandbox phone number', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleDeleteSmsSandboxNumber = async (phoneNumber: string) => {
    try {
      await api.deleteSMSSandboxPhoneNumber(phoneNumber, region)
      await fetchSmsData()
      toast.success('SMS sandbox phone number deleted')
    } catch (error) {
      toast.error('Failed to delete SMS sandbox phone number', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleCheckOptedOut = async () => {
    if (!checkOptedOutPhoneNumber.trim()) return
    try {
      const optedOut = await api.checkIfPhoneNumberIsOptedOut(checkOptedOutPhoneNumber.trim(), region)
      toast.info(optedOut ? 'Phone number is opted out' : 'Phone number is not opted out')
    } catch (error) {
      toast.error('Failed to check phone number', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleOptInPhoneNumber = async () => {
    if (!optInPhoneNumber.trim()) return
    try {
      await api.optInPhoneNumber(optInPhoneNumber.trim(), region)
      setOptInPhoneNumber('')
      await fetchSmsData()
      toast.success('Phone number opted in')
    } catch (error) {
      toast.error('Failed to opt in phone number', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const handleDeleteOptedOut = async () => {
    if (!optOutPhoneNumber.trim()) return
    try {
      await api.optInPhoneNumber(optOutPhoneNumber.trim(), region)
      setOptOutPhoneNumber('')
      await fetchSmsData()
      toast.success('Phone number removed from opt-out list')
    } catch (error) {
      toast.error('Failed to update opt-out phone number', { description: error instanceof Error ? error.message : String(error) })
    }
  }

  const renderTopicSection = () => {
    if (currentTopicName) {
      return (
        <div className="flex h-full flex-col gap-4 p-4">
          <div className="rounded-xl border bg-card p-4 shadow-xs/5">
            <div className="flex items-center justify-between gap-2">
              <div>
                <h3 className="text-sm font-semibold">Topic details</h3>
                <p className="text-xs text-muted-foreground">
                  {currentTopicArn ? topicNameFromArn(currentTopicArn) : `Loading ${currentTopicName}...`}
                </p>
              </div>
              {currentTopicArn && <Badge variant="outline">Selected</Badge>}
            </div>

            {loading || topicDetailsLoading ? (
              <div className="flex h-40 items-center justify-center">
                <Spinner className="h-6 w-6 text-muted-foreground" />
              </div>
            ) : !currentTopicArn ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Layers3 className="h-6 w-6" />
                  </EmptyMedia>
                  <EmptyTitle>Topic not found</EmptyTitle>
                  <EmptyDescription>
                    {`No SNS topic matched "${currentTopicName}".`}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <Tabs defaultValue="overview" className="flex flex-col gap-4">
                <TabsList className="w-full justify-start">
                  <TabsTrigger value="overview">Overview</TabsTrigger>
                  <TabsTrigger value="publish">Publish</TabsTrigger>
                  <TabsTrigger value="subscriptions">Subscriptions</TabsTrigger>
                  <TabsTrigger value="admin">Admin</TabsTrigger>
                </TabsList>

                <TabsContent value="overview" className="space-y-4">
                  <div className="rounded-lg border p-3 text-xs">
                    <div className="font-mono break-all">{currentTopicArn}</div>
                  </div>
                  <div className="grid gap-2">
                    {Object.entries(topicAttributes).map(([key, value]) => (
                      <div key={key} className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-xs">
                        <span className="font-medium">{key}</span>
                        <span className="break-all text-muted-foreground">{value}</span>
                      </div>
                    ))}
                  </div>

                  <div className="grid gap-2 md:grid-cols-3">
                    <Input value={topicAttributeName} onChange={(e) => setTopicAttributeName(e.target.value)} placeholder="Attribute name" />
                    <Input value={topicAttributeValue} onChange={(e) => setTopicAttributeValue(e.target.value)} placeholder="Attribute value" />
                    <Button onClick={handleSetTopicAttribute} disabled={!topicAttributeName.trim()}>
                      <RefreshCw className="h-4 w-4" />
                      Save Attribute
                    </Button>
                  </div>
                </TabsContent>

                <TabsContent value="publish" className="space-y-4">
                  <Textarea value={topicMessage} onChange={(e) => setTopicMessage(e.target.value)} rows={5} />
                  <Input value={topicSubject} onChange={(e) => setTopicSubject(e.target.value)} placeholder="Subject" />
                  <div className="flex flex-wrap gap-2">
                    <Button onClick={handlePublishTopic} disabled={!topicMessage.trim()}>
                      <Send className="h-4 w-4" />
                      Publish
                    </Button>
                    <Button variant="outline" onClick={handlePublishBatch}>
                      <MessageSquareText className="h-4 w-4" />
                      Publish Batch
                    </Button>
                  </div>
                  <div className="grid gap-2 md:grid-cols-2">
                    <Textarea value={topicBatchMessageOne} onChange={(e) => setTopicBatchMessageOne(e.target.value)} rows={4} />
                    <Textarea value={topicBatchMessageTwo} onChange={(e) => setTopicBatchMessageTwo(e.target.value)} rows={4} />
                  </div>
                  {topicPublishStatus && <div className="text-xs text-muted-foreground">{topicPublishStatus}</div>}
                </TabsContent>

                <TabsContent value="subscriptions" className="space-y-4">
                  <div className="grid gap-2 md:grid-cols-3">
                    <Input value={topicTagKey} onChange={(e) => setTopicTagKey(e.target.value)} placeholder="Tag key" />
                    <Input value={topicTagValue} onChange={(e) => setTopicTagValue(e.target.value)} placeholder="Tag value" />
                    <Button onClick={handleAddTopicTag} disabled={!topicTagKey.trim()}>
                      <Plus className="h-4 w-4" />
                      Add Tag
                    </Button>
                  </div>

                  <div className="space-y-2">
                    {topicSubscriptions.map((subscription) => (
                      <div key={subscription.SubscriptionArn} className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-xs">
                        <div className="min-w-0">
                          <div className="font-medium">{subscription.Protocol}</div>
                          <div className="truncate text-muted-foreground">{subscription.Endpoint}</div>
                        </div>
                        <Button variant="ghost" size="icon-sm" onClick={() => void handleDeleteSubscription(subscription.SubscriptionArn)}>
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="admin" className="space-y-4">
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <h4 className="text-sm font-medium">Tags</h4>
                      <Badge variant="outline">{topicTags.length}</Badge>
                    </div>
                    {topicTags.map((tag) => (
                      <div key={tag.Key} className="flex items-center justify-between rounded-lg border px-3 py-2 text-xs">
                        <div>
                          <div className="font-medium">{tag.Key}</div>
                          <div className="text-muted-foreground">{tag.Value}</div>
                        </div>
                        <Button variant="ghost" size="icon-sm" onClick={() => tag.Key && void handleRemoveTopicTag(tag.Key)}>
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    ))}
                  </div>

                  <Separator />

                  <div className="space-y-2">
                    <h4 className="text-sm font-medium">Data protection policy</h4>
                    <Textarea value={topicPolicy} onChange={(e) => setTopicPolicy(e.target.value)} rows={8} />
                    <Button variant="outline" onClick={handleSavePolicy}>
                      <ShieldCheck className="h-4 w-4" />
                      Save Policy
                    </Button>
                  </div>

                  <Separator />

                  <div className="space-y-2">
                    <h4 className="text-sm font-medium">Permissions</h4>
                    <div className="grid gap-2 md:grid-cols-3">
                      <Input value={topicPermissionLabel} onChange={(e) => setTopicPermissionLabel(e.target.value)} placeholder="Label" />
                      <Input value={topicPermissionAccounts} onChange={(e) => setTopicPermissionAccounts(e.target.value)} placeholder="AWS accounts, comma separated" />
                      <Input value={topicPermissionActions} onChange={(e) => setTopicPermissionActions(e.target.value)} placeholder="Actions, comma separated" />
                    </div>
                    <div className="flex gap-2">
                      <Button onClick={handleAddPermission}>Add Permission</Button>
                      <Button variant="outline" onClick={handleRemovePermission}>Remove Permission</Button>
                    </div>
                  </div>
                </TabsContent>
              </Tabs>
            )}
          </div>
        </div>
      )
    }

    return (
      <div className="flex h-full flex-col gap-4 p-4">
        <div className="flex flex-col gap-3 rounded-xl border bg-card p-4 shadow-xs/5">
          <div className="relative w-full">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search topics..."
              className="pl-10"
            />
          </div>

          <Dialog
            open={isCreateTopicOpen}
            onOpenChange={(open) => {
              setIsCreateTopicOpen(open)
              if (!open) {
                resetCreateTopicForm()
              }
            }}
          >
            <div className="flex justify-end">
              <DialogTrigger asChild>
                <Button variant="outline" className="w-full sm:w-auto">
                  <Plus className="h-4 w-4" />
                  Create Topic
                </Button>
              </DialogTrigger>
            </div>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Topic</DialogTitle>
                <DialogDescription>
                  Create a new SNS topic in your local environment.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 px-6 pb-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="topic-name">
                    Topic name
                  </label>
                  <Input
                    id="topic-name"
                    value={newTopicName}
                    onChange={(e) => setNewTopicName(e.target.value)}
                    placeholder="my-topic"
                    autoFocus
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="topic-attributes">
                    Attributes
                  </label>
                  <Input
                    id="topic-attributes"
                    value={newTopicAttributes}
                    onChange={(e) => setNewTopicAttributes(e.target.value)}
                    placeholder='{"DisplayName":"my-topic"}'
                  />
                  <p className="text-xs text-muted-foreground">
                    Optional JSON object. Leave blank to use the default topic attributes.
                  </p>
                </div>
              </div>
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="ghost" onClick={resetCreateTopicForm}>
                    Cancel
                  </Button>
                </DialogClose>
                <Button onClick={handleCreateTopic} disabled={!newTopicName.trim() || isCreatingTopic}>
                  <Plus className="h-4 w-4" />
                  {isCreatingTopic ? 'Creating...' : 'Create Topic'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>

        {loading ? (
          <div className="flex h-40 items-center justify-center">
            <Spinner className="h-6 w-6 text-muted-foreground" />
          </div>
        ) : filteredTopics.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Bell className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>No topics found</EmptyTitle>
              <EmptyDescription>
                {search ? `No topics match "${search}"` : 'Create your first SNS topic to get started.'}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className="grid gap-4">
            {filteredTopics.map((topic) => (
              <SpotlightCard
                key={topic.TopicArn}
                className="cursor-pointer transition-colors"
                onClick={() => navigate(`/resources/sns/${encodeURIComponent(topic.TopicName)}`)}
              >
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    {topic.TopicName}
                    <Badge variant="outline">{topic.Tags.length} tags</Badge>
                  </CardTitle>
                  <CardDescription className="break-all">{topic.TopicArn}</CardDescription>
                  <CardAction>
                    <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); void handleDeleteTopic(topic.TopicArn) }}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </CardAction>
                </CardHeader>
              </SpotlightCard>
            ))}
          </div>
        )}
      </div>
    )
  }

  const renderSubscriptionsSection = () => (
    <div className="flex h-full flex-col gap-4 p-4">
      <div className="flex flex-col gap-3 rounded-xl border bg-card p-4 shadow-xs/5 lg:flex-row lg:items-center lg:justify-between">
        <div className="relative w-full lg:max-w-md">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search subscriptions..." className="pl-10" />
        </div>
        <div className="grid gap-2 lg:grid-cols-3">
          <Input value={subscriptionTopicArn} onChange={(e) => setSubscriptionTopicArn(e.target.value)} placeholder="Topic ARN" />
          <Input value={subscriptionProtocol} onChange={(e) => setSubscriptionProtocol(e.target.value)} placeholder="Protocol" />
          <Input value={subscriptionEndpoint} onChange={(e) => setSubscriptionEndpoint(e.target.value)} placeholder="Endpoint" />
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.1fr_1fr]">
        <div className="space-y-2">
          <Button onClick={handleSubscribe}>
            <Plus className="h-4 w-4" />
            Create Subscription
          </Button>
          <div className="flex gap-2">
            <Input value={subscriptionToken} onChange={(e) => setSubscriptionToken(e.target.value)} placeholder="Confirmation token" />
            <Button variant="outline" onClick={handleConfirmSubscription}>Confirm</Button>
          </div>

          {loading ? (
            <div className="flex h-40 items-center justify-center"><Spinner className="h-6 w-6 text-muted-foreground" /></div>
          ) : filteredSubscriptions.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Users className="h-6 w-6" />
                </EmptyMedia>
                <EmptyTitle>No subscriptions found</EmptyTitle>
                <EmptyDescription>{search ? `No subscriptions match "${search}"` : 'Create your first subscription to route messages.'}</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="grid gap-3">
              {filteredSubscriptions.map((subscription) => (
                <SpotlightCard
                  key={subscription.SubscriptionArn}
                  className={`cursor-pointer transition-colors ${selectedSubscriptionArn === subscription.SubscriptionArn ? 'border-primary/40 bg-primary/5' : ''}`}
                  onClick={() => setSelectedSubscriptionArn(subscription.SubscriptionArn)}
                >
                  <CardHeader>
                    <CardTitle className="text-sm">{subscription.Protocol}</CardTitle>
                    <CardDescription className="break-all">{subscription.Endpoint}</CardDescription>
                    <CardAction>
                      <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); void handleDeleteSubscription(subscription.SubscriptionArn) }}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </CardAction>
                  </CardHeader>
                </SpotlightCard>
              ))}
            </div>
          )}
        </div>

        <div className="space-y-4 rounded-xl border bg-card p-4 shadow-xs/5">
          <h3 className="text-sm font-semibold">Subscription details</h3>
          {!selectedSubscriptionArn ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Users className="h-6 w-6" />
                </EmptyMedia>
                <EmptyTitle>No subscription selected</EmptyTitle>
                <EmptyDescription>Select a subscription to inspect its attributes.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="space-y-3">
              <div className="rounded-lg border p-3 text-xs font-mono break-all">{selectedSubscriptionArn}</div>
              {Object.entries(subscriptionAttributes).map(([key, value]) => (
                <div key={key} className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-xs">
                  <span className="font-medium">{key}</span>
                  <span className="break-all text-muted-foreground">{value}</span>
                </div>
              ))}
              <div className="grid gap-2 md:grid-cols-3">
                <Input value={subscriptionAttributeName} onChange={(e) => setSubscriptionAttributeName(e.target.value)} placeholder="Attribute name" />
                <Input value={subscriptionAttributeValue} onChange={(e) => setSubscriptionAttributeValue(e.target.value)} placeholder="Attribute value" />
                <Button onClick={handleSetSubscriptionAttribute}>Save Attribute</Button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )

  const renderPlatformApplicationsSection = () => (
    <div className="flex h-full flex-col gap-4 p-4">
      <div className="flex flex-col gap-3 rounded-xl border bg-card p-4 shadow-xs/5 lg:flex-row lg:items-center lg:justify-between">
        <div className="relative w-full lg:max-w-md">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search platform applications..." className="pl-10" />
        </div>
        <div className="grid gap-2 lg:grid-cols-3">
          <Input value={newPlatformAppName} onChange={(e) => setNewPlatformAppName(e.target.value)} placeholder="Application name" />
          <Input value={newPlatformAppPlatform} onChange={(e) => setNewPlatformAppPlatform(e.target.value)} placeholder="Platform (APNS, GCM, etc.)" />
          <Input value={newPlatformAppAttributes} onChange={(e) => setNewPlatformAppAttributes(e.target.value)} placeholder="Attributes JSON" />
        </div>
        <Button onClick={handleCreatePlatformApp} disabled={!newPlatformAppName.trim()}>
          <Plus className="h-4 w-4" />
          Create App
        </Button>
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.1fr_1fr]">
        <div className="space-y-2">
          {loading ? (
            <div className="flex h-40 items-center justify-center"><Spinner className="h-6 w-6 text-muted-foreground" /></div>
          ) : filteredApps.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Smartphone className="h-6 w-6" />
                </EmptyMedia>
                <EmptyTitle>No platform applications found</EmptyTitle>
                <EmptyDescription>{search ? `No apps match "${search}"` : 'Create a platform app to manage push endpoints.'}</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="grid gap-3">
              {filteredApps.map((app) => (
                <SpotlightCard
                  key={app.PlatformApplicationArn}
                  className={`cursor-pointer transition-colors ${selectedAppArn === app.PlatformApplicationArn ? 'border-primary/40 bg-primary/5' : ''}`}
                  onClick={() => setSelectedAppArn(app.PlatformApplicationArn)}
                >
                  <CardHeader>
                    <CardTitle className="text-sm">{app.Name}</CardTitle>
                    <CardDescription className="break-all">{app.PlatformApplicationArn}</CardDescription>
                    <CardAction>
                      <Button variant="ghost" size="icon-sm" onClick={(e) => { e.stopPropagation(); void handleDeletePlatformApp(app.PlatformApplicationArn) }}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </CardAction>
                  </CardHeader>
                </SpotlightCard>
              ))}
            </div>
          )}
        </div>

        <div className="space-y-4 rounded-xl border bg-card p-4 shadow-xs/5">
          <h3 className="text-sm font-semibold">Platform application details</h3>
          {!selectedAppArn ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Smartphone className="h-6 w-6" />
                </EmptyMedia>
                <EmptyTitle>No platform app selected</EmptyTitle>
                <EmptyDescription>Select one to inspect endpoints and attributes.</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="space-y-4">
              <div className="rounded-lg border p-3 text-xs font-mono break-all">{selectedAppArn}</div>
              <div className="grid gap-2">
                {Object.entries(platformAppAttributes).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-xs">
                    <span className="font-medium">{key}</span>
                    <span className="break-all text-muted-foreground">{value}</span>
                  </div>
                ))}
              </div>
              <div className="grid gap-2 md:grid-cols-3">
                <Input value={platformAttributeName} onChange={(e) => setPlatformAttributeName(e.target.value)} placeholder="Attribute name" />
                <Input value={platformAttributeValue} onChange={(e) => setPlatformAttributeValue(e.target.value)} placeholder="Attribute value" />
                <Button onClick={handleSetPlatformAppAttribute}>Save Attribute</Button>
              </div>
              <Separator />
              <div className="grid gap-2 md:grid-cols-3">
                <Input value={newEndpointToken} onChange={(e) => setNewEndpointToken(e.target.value)} placeholder="Device token" />
                <Input value={newEndpointCustomUserData} onChange={(e) => setNewEndpointCustomUserData(e.target.value)} placeholder="Custom user data" />
                <Input value={newEndpointAttributes} onChange={(e) => setNewEndpointAttributes(e.target.value)} placeholder="Endpoint attributes JSON" />
              </div>
              <Button onClick={handleCreateEndpoint} disabled={!newEndpointToken.trim()}>
                <Plus className="h-4 w-4" />
                Create Endpoint
              </Button>
              <div className="space-y-2">
                {platformEndpoints.map((endpoint) => (
                  <div key={endpoint.EndpointArn} className="flex items-start justify-between gap-3 rounded-lg border px-3 py-2 text-xs">
                    <div className="min-w-0">
                      <div className="font-mono break-all">{endpoint.EndpointArn}</div>
                      <div className="mt-1 text-muted-foreground">{formatAttributesJson(endpoint.Attributes)}</div>
                    </div>
                    <Button variant="ghost" size="icon-sm" onClick={() => void handleDeleteEndpoint(endpoint.EndpointArn)}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )

  const renderSmsSection = () => (
    <div className="flex h-full flex-col gap-4 p-4">
      <div className="grid gap-4 xl:grid-cols-[1.1fr_1fr]">
        <div className="space-y-4">
          <div className="flex flex-col gap-3 rounded-xl border bg-card p-4 shadow-xs/5">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={smsSandbox ? 'destructive' : 'outline'}>
                {smsSandbox ? 'Sandbox mode' : 'Production mode'}
              </Badge>
              <Badge variant="outline">{smsNumbers.length} sandbox numbers</Badge>
              <Badge variant="outline">{optedOutNumbers.length} opted out</Badge>
              <Badge variant="outline">{originationNumbers.length} origination numbers</Badge>
            </div>
            <div className="grid gap-2 md:grid-cols-3">
              <Input value={smsAttributeName} onChange={(e) => setSmsAttributeName(e.target.value)} placeholder="SMS attribute name" />
              <Input value={smsAttributeValue} onChange={(e) => setSmsAttributeValue(e.target.value)} placeholder="SMS attribute value" />
              <Button onClick={handleSetSmsAttributes}>Save SMS Attribute</Button>
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <SpotlightCard className="space-y-3 p-4">
              <CardTitle className="text-sm">Add sandbox number</CardTitle>
              <Input value={newSmsPhoneNumber} onChange={(e) => setNewSmsPhoneNumber(e.target.value)} placeholder="+15555550100" />
              <Input value={newSmsLanguageCode} onChange={(e) => setNewSmsLanguageCode(e.target.value)} placeholder="Language code" />
              <Button onClick={handleCreateSmsSandboxNumber} disabled={!newSmsPhoneNumber.trim()}>
                <Plus className="h-4 w-4" />
                Add number
              </Button>
            </SpotlightCard>

            <SpotlightCard className="space-y-3 p-4">
              <CardTitle className="text-sm">Verify sandbox number</CardTitle>
              <Input value={verifySmsPhoneNumber} onChange={(e) => setVerifySmsPhoneNumber(e.target.value)} placeholder="+15555550100" />
              <Input value={verifySmsOtp} onChange={(e) => setVerifySmsOtp(e.target.value)} placeholder="OTP" />
              <Button onClick={handleVerifySmsSandboxNumber} disabled={!verifySmsPhoneNumber.trim() || !verifySmsOtp.trim()}>
                <RefreshCw className="h-4 w-4" />
                Verify
              </Button>
            </SpotlightCard>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <SpotlightCard className="space-y-3 p-4">
              <CardTitle className="text-sm">Opt in / out</CardTitle>
              <Input value={checkOptedOutPhoneNumber} onChange={(e) => setCheckOptedOutPhoneNumber(e.target.value)} placeholder="Check phone number" />
              <div className="flex gap-2">
                <Button variant="outline" onClick={handleCheckOptedOut}>Check</Button>
              </div>
              <Input value={optInPhoneNumber} onChange={(e) => setOptInPhoneNumber(e.target.value)} placeholder="Opt in number" />
              <Button onClick={handleOptInPhoneNumber} disabled={!optInPhoneNumber.trim()}>Opt In</Button>
              <Input value={optOutPhoneNumber} onChange={(e) => setOptOutPhoneNumber(e.target.value)} placeholder="Remove from opt-out list" />
              <Button variant="outline" onClick={handleDeleteOptedOut} disabled={!optOutPhoneNumber.trim()}>Remove Opt-out</Button>
            </SpotlightCard>

            <SpotlightCard className="space-y-3 p-4">
              <CardTitle className="text-sm">SMS attributes</CardTitle>
              <div className="rounded-lg border p-3 text-xs">{formatAttributesJson(smsAttributes)}</div>
            </SpotlightCard>
          </div>
        </div>

        <div className="space-y-4 rounded-xl border bg-card p-4 shadow-xs/5">
          <h3 className="text-sm font-semibold">Sandbox inventory</h3>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Sandbox numbers</h4>
              <Badge variant="outline">{filteredSmsNumbers.length}</Badge>
            </div>
            {filteredSmsNumbers.map((entry) => (
              <div key={entry.PhoneNumber} className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-xs">
                <div>
                  <div className="font-medium">{entry.PhoneNumber}</div>
                  <div className="text-muted-foreground">{entry.Status ?? 'Unknown'} {entry.LanguageCode ? `· ${entry.LanguageCode}` : ''}</div>
                </div>
                <Button variant="ghost" size="icon-sm" onClick={() => void handleDeleteSmsSandboxNumber(entry.PhoneNumber)}>
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ))}
          </div>

          <Separator />

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Opted out</h4>
              <Badge variant="outline">{optedOutNumbers.length}</Badge>
            </div>
            {optedOutNumbers.map((phoneNumber) => (
              <div key={phoneNumber} className="rounded-lg border px-3 py-2 text-xs">{phoneNumber}</div>
            ))}
          </div>

          <Separator />

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">Origination numbers</h4>
              <Badge variant="outline">{originationNumbers.length}</Badge>
            </div>
            {originationNumbers.map((entry) => (
              <div key={entry.PhoneNumber} className="rounded-lg border px-3 py-2 text-xs">
                <div className="font-medium">{entry.PhoneNumber}</div>
                <div className="text-muted-foreground">{entry.Status ?? 'Unknown'}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )

  return (
    <Tabs
      value={activeSection}
      onValueChange={(value) => {
        const section = value as SNSBrowserSection
        setActiveSection(section)
        setSearchParams(section === 'topics' ? {} : { section })
      }}
      className="flex h-full flex-col"
    >
      <div className="flex-none border-b px-4 py-3">
        <TabsList className="w-full justify-start">
          <TabsTrigger value="topics">Topics</TabsTrigger>
          <TabsTrigger value="subscriptions">Subscriptions</TabsTrigger>
          <TabsTrigger value="platform-applications">Platform Apps</TabsTrigger>
          <TabsTrigger value="sms">SMS Sandbox</TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value="topics" className="min-h-0 flex-1">
        {renderTopicSection()}
      </TabsContent>
      <TabsContent value="subscriptions" className="min-h-0 flex-1">
        {renderSubscriptionsSection()}
      </TabsContent>
      <TabsContent value="platform-applications" className="min-h-0 flex-1">
        {renderPlatformApplicationsSection()}
      </TabsContent>
      <TabsContent value="sms" className="min-h-0 flex-1">
        {renderSmsSection()}
      </TabsContent>
    </Tabs>
  )
}
