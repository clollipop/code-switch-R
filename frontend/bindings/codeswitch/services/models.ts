// Shared models for services

// 环境变量冲突
export interface EnvConflict {
  varName: string
  varValue: string
  sourceType: 'system' | 'file'
  sourcePath: string
}

// 自定义提示词
export interface Prompt {
  id: string
  name: string
  content: string
  description?: string
  enabled: boolean
  createdAt?: number
  updatedAt?: number
}

// 端点延迟测试结果
export interface EndpointLatency {
  url: string                // 端点 URL
  latency: number | null     // 延迟（毫秒），null 表示失败
  status?: number            // HTTP 状态码
  error?: string             // 错误信息
}
