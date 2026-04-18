import { Frame, FrameDescription, FrameHeader, FramePanel, FrameTitle } from "@renderer/components/ui/frame"
import { ServiceCard } from "./components/service-card"
import { Search } from "lucide-react"
import { useState } from "react"
import { Separator } from "@renderer/components/ui/separator"
import { RESOURCES, COMING_SOON_RESOURCES } from "./constants"
import { useNavigate } from "react-router"

function ResourcesPage(): React.JSX.Element {
  const [searchQuery, setSearchQuery] = useState("")
  const navigate = useNavigate()

  const filteredResources = RESOURCES.filter(resource =>
    resource.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    resource.description.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const filteredSoon = COMING_SOON_RESOURCES.filter(resource =>
    resource.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    resource.description.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const handleResourceClick = (title: string) => {
    if (title === "S3") {
      navigate('/resources/s3')
    }
  }

  return (
    <Frame className="w-full">
      <FrameHeader>
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 w-full">
          <div>
            <FrameTitle>Resources</FrameTitle>
            <FrameDescription>Browse and manage MildStack resources</FrameDescription>
          </div>
          
          <div className="relative group max-w-sm w-full">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-neutral-500 group-focus-within:text-neutral-300 transition-colors" />
            <input
              type="text"
              placeholder="Search resources..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full h-10 pl-10 pr-4 rounded-xl border border-neutral-800 bg-neutral-900/50 text-sm text-neutral-100 placeholder:text-neutral-500 focus:outline-none focus:ring-2 focus:ring-neutral-700 transition-all"
            />
          </div>
        </div>
      </FrameHeader>

      <FramePanel className="flex-1 overflow-y-auto border-none bg-transparent shadow-none p-0">
        <div className="space-y-8 px-4 pb-8">
          {/* Available Resources */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredResources.map((resource) => (
              <ServiceCard
                key={resource.title}
                title={resource.title}
                description={resource.description}
                icon={resource.icon}
                color={resource.color}
                onClick={() => handleResourceClick(resource.title)}
              />
            ))}
          </div>

          {/* Coming Soon Section */}
          {(filteredSoon.length > 0) && (
            <div className="space-y-4">
              <div className="flex items-center gap-4">
                <h2 className="text-sm font-semibold text-neutral-500 uppercase tracking-widest shrink-0">
                  Coming Soon
                </h2>
                <Separator className="bg-neutral-800/50 flex-1" />
              </div>
              
              <div className="bg-neutral-900/30 border border-neutral-800/50 rounded-2xl p-6 text-center">
                <p className="text-sm text-neutral-400 italic">
                  We're expanding the MildStack! New services are being developed to bring the full AWS experience to your local environment.
                </p>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {filteredSoon.map((resource) => (
                  <ServiceCard
                    key={resource.title}
                    title={resource.title}
                    description={resource.description}
                    icon={resource.icon}
                    color={resource.color}
                    disabled
                  />
                ))}
              </div>
            </div>
          )}
        </div>
        
        {filteredResources.length === 0 && filteredSoon.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-neutral-500">
            <Search size={48} className="mb-4 opacity-20" />
            <p className="text-lg text-center px-4">No resources found matching "{searchQuery}"</p>
          </div>
        )}
      </FramePanel>
    </Frame>
  )
}

export default ResourcesPage

