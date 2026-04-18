import React from 'react'
import { cn } from '@renderer/lib/utils'
import SpotlightCard from '../../../components/ui/spotlight-card'
import type { LucideIcon } from 'lucide-react'

interface ServiceCardProps {
  title: string
  description: string
  icon: LucideIcon
  color?: string
  disabled?: boolean
  onClick?: () => void
}

const ServiceCard: React.FC<ServiceCardProps> = ({ title, description, icon: Icon, color, disabled, onClick }) => {
  return (
    <SpotlightCard 
      className={cn(
        "group border-neutral-800/50 bg-neutral-900/50 transition-all duration-300 p-5",
        disabled ? "opacity-50 cursor-not-allowed grayscale-[0.5]" : "cursor-pointer hover:border-neutral-700/50"
      )}
      spotlightColor={disabled ? "rgba(255, 255, 255, 0.05)" : undefined}
      onClick={disabled ? undefined : onClick}
    >
      <div className="flex flex-row items-center gap-4">
        <div 
          className={cn(
            "w-10 h-10 shrink-0 rounded-lg flex items-center justify-center transition-colors duration-300",
            disabled ? "bg-neutral-900" : "bg-neutral-800 group-hover:bg-neutral-700"
          )}
          style={{ color: disabled ? '#666' : (color || '#fff') }}
        >
          <Icon size={20} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className={cn(
              "text-base font-semibold transition-colors truncate",
              disabled ? "text-neutral-500" : "text-neutral-100 group-hover:text-white"
            )}>
              {title}
            </h3>
            {disabled && (
              <span className="text-[10px] px-1.5 py-0.5 rounded-md bg-neutral-800 text-neutral-500 font-medium uppercase tracking-wider">
                Soon
              </span>
            )}
          </div>
          <p className={cn(
            "text-xs line-clamp-1 leading-relaxed",
            disabled ? "text-neutral-600" : "text-neutral-400"
          )}>
            {description}
          </p>
        </div>
      </div>
    </SpotlightCard>
  )
}

export { ServiceCard }