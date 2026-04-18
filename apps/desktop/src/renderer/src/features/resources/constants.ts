import { Database, MessageSquare, Zap, Bell, Activity, HardDrive, Cloud, Globe, Layers } from "lucide-react"
import type { LucideIcon } from "lucide-react"

export interface ResourceItem {
  title: string
  description: string
  icon: LucideIcon
  color: string
}

export const RESOURCES: ResourceItem[] = [
  {
    title: "S3",
    description: "Scalable object storage for data, backups, and static assets.",
    icon: HardDrive,
    color: "#FF9900"
  },
  {
    title: "SQS",
    description: "Fully managed message queuing for microservices and distributed systems.",
    icon: MessageSquare,
    color: "#FF4F8B"
  },
  {
    title: "DynamoDB",
    description: "Fast, flexible NoSQL database for any scale.",
    icon: Database,
    color: "#402770"
  },
  {
    title: "Lambda",
    description: "Serverless compute to run code without provisioning servers.",
    icon: Zap,
    color: "#FF9900"
  },
  {
    title: "SNS",
    description: "Pub/sub messaging and mobile notifications service.",
    icon: Bell,
    color: "#FF4F8B"
  },
  {
    title: "CloudWatch",
    description: "Monitoring and observability for your AWS resources and apps.",
    icon: Activity,
    color: "#FF4F8B"
  }
]

export const COMING_SOON_RESOURCES: ResourceItem[] = [
  {
    title: "RDS",
    description: "Relational database service for MySQL, PostgreSQL, and more.",
    icon: Database,
    color: "#3B48CC"
  },
  {
    title: "ECS",
    description: "Highly scalable, high-performance container management service.",
    icon: Layers,
    color: "#FF9900"
  },
  {
    title: "ElastiCache",
    description: "In-memory data store and cache service for Redis or Memcached.",
    icon: Cloud,
    color: "#C925D1"
  },
  {
    title: "Route53",
    description: "Scalable Domain Name System (DNS) web service.",
    icon: Globe,
    color: "#FF9900"
  }
]
