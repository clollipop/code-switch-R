// This file is auto-generated. DO NOT EDIT.
import { Call } from '@wailsio/runtime'

export interface Provider {
  id: number
  name: string
  apiUrl: string
  apiKey: string
  officialSite?: string
  icon?: string
  tint?: string
  accent?: string
  enabled: boolean
  supportedModels?: Record<string, boolean>
  modelMapping?: Record<string, string>
  level?: number
}

export function LoadProviders(kind: string): Promise<Provider[]> {
  return Call.ByName('codeswitch/services.ProviderService.LoadProviders', kind)
}

export function SaveProviders(kind: string, providers: Provider[]): Promise<void> {
  return Call.ByName('codeswitch/services.ProviderService.SaveProviders', kind, providers)
}

export function DuplicateProvider(kind: string, sourceID: number): Promise<Provider> {
  return Call.ByName('codeswitch/services.ProviderService.DuplicateProvider', kind, sourceID)
}
