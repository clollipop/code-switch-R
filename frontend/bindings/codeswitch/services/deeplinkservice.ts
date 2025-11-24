// TypeScript bindings for DeepLinkService
import { Call } from '@wailsio/runtime'

export interface DeepLinkImportRequest {
  version: string
  resource: string
  app: string
  name: string
  homepage: string
  endpoint: string
  apiKey: string
  model?: string
  notes?: string
  haikuModel?: string
  sonnetModel?: string
  opusModel?: string
  config?: string
  configFormat?: string
  configUrl?: string
}

/**
 * ParseDeepLinkURL 解析 ccswitch:// URL
 */
export function ParseDeepLinkURL(urlStr: string): Promise<DeepLinkImportRequest> {
  return Call.ByName('codeswitch/services.DeepLinkService.ParseDeepLinkURL', urlStr)
}

/**
 * ImportProviderFromDeepLink 从深度链接导入供应商
 */
export function ImportProviderFromDeepLink(request: DeepLinkImportRequest): Promise<string> {
  return Call.ByName('codeswitch/services.DeepLinkService.ImportProviderFromDeepLink', request)
}
