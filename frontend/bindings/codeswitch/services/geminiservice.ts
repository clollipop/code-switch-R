// Gemini Service TypeScript bindings
import { Call } from '@wailsio/runtime'

// Gemini 认证类型
export type GeminiAuthType = 'oauth-personal' | 'gemini-api-key' | 'packycode' | 'generic'

// Gemini 供应商配置
export interface GeminiProvider {
  id: string
  name: string
  websiteUrl?: string
  apiKeyUrl?: string
  baseUrl?: string
  apiKey?: string
  model?: string
  description?: string
  category?: string // official, third_party, custom
  partnerPromotionKey?: string
  enabled: boolean
  level?: number // 优先级分组 (1-10, 默认 1)
  envConfig?: Record<string, string>
  settingsConfig?: Record<string, any>
}

// Gemini 预设供应商
export interface GeminiPreset {
  name: string
  websiteUrl: string
  apiKeyUrl?: string
  baseUrl?: string
  model?: string
  description?: string
  category: string
  partnerPromotionKey?: string
  envConfig?: Record<string, string>
}

// Gemini 配置状态
export interface GeminiStatus {
  enabled: boolean
  currentProvider?: string
  authType: GeminiAuthType
  hasApiKey: boolean
  hasBaseUrl: boolean
  model?: string
}

// 获取预设供应商列表
export function GetPresets(): Promise<GeminiPreset[]> {
  return Call.ByName('codeswitch/services.GeminiService.GetPresets')
}

// 获取已配置的供应商列表
export function GetProviders(): Promise<GeminiProvider[]> {
  return Call.ByName('codeswitch/services.GeminiService.GetProviders')
}

// 添加供应商
export function AddProvider(provider: GeminiProvider): Promise<void> {
  return Call.ByName('codeswitch/services.GeminiService.AddProvider', provider)
}

// 更新供应商
export function UpdateProvider(provider: GeminiProvider): Promise<void> {
  return Call.ByName('codeswitch/services.GeminiService.UpdateProvider', provider)
}

// 删除供应商
export function DeleteProvider(id: string): Promise<void> {
  return Call.ByName('codeswitch/services.GeminiService.DeleteProvider', id)
}

// 切换到指定供应商
export function SwitchProvider(id: string): Promise<void> {
  return Call.ByName('codeswitch/services.GeminiService.SwitchProvider', id)
}

// 获取当前 Gemini 配置状态
export function GetStatus(): Promise<GeminiStatus> {
  return Call.ByName('codeswitch/services.GeminiService.GetStatus')
}

// 从预设创建供应商
export function CreateProviderFromPreset(presetName: string, apiKey: string): Promise<GeminiProvider> {
  return Call.ByName('codeswitch/services.GeminiService.CreateProviderFromPreset', presetName, apiKey)
}

// 重新排序供应商（按传入的 ID 顺序）
export function ReorderProviders(ids: string[]): Promise<void> {
  return Call.ByName('codeswitch/services.GeminiService.ReorderProviders', ids)
}
