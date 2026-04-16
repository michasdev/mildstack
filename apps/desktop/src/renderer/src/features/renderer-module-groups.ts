import type { RendererModuleGroup } from '@renderer/shared/types/renderer-module-group'

export const rendererModuleGroups = [
  'app',
  'features',
  'shared'
] as const satisfies readonly RendererModuleGroup[]
