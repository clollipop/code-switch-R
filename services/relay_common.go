package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ============================================================================
// 通用接口定义
// ============================================================================

// ProviderLike 是所有 Provider 类型的通用接口
// 通过接口抽象，我们可以用同一套代码处理 Provider 和 GeminiProvider
type ProviderLike interface {
	GetName() string
	GetLevel() int
	IsEnabled() bool
	HasValidConfig() bool
}

// 确保 Provider 实现 ProviderLike 接口
var _ ProviderLike = Provider{}

// GetName 返回 Provider 名称
func (p Provider) GetName() string { return p.Name }

// GetLevel 返回 Provider 级别，未配置时返回 1
func (p Provider) GetLevel() int {
	if p.Level <= 0 {
		return 1
	}
	return p.Level
}

// IsEnabled 返回 Provider 是否启用
func (p Provider) IsEnabled() bool { return p.Enabled }

// HasValidConfig 检查 Provider 是否有有效配置
func (p Provider) HasValidConfig() bool {
	return p.APIURL != "" && p.APIKey != ""
}

// 确保 GeminiProvider 实现 ProviderLike 接口
var _ ProviderLike = GeminiProvider{}

// GetName 返回 GeminiProvider 名称
func (p GeminiProvider) GetName() string { return p.Name }

// GetLevel 返回 GeminiProvider 级别，未配置时返回 1
func (p GeminiProvider) GetLevel() int {
	if p.Level <= 0 {
		return 1
	}
	return p.Level
}

// IsEnabled 返回 GeminiProvider 是否启用
func (p GeminiProvider) IsEnabled() bool { return p.Enabled }

// HasValidConfig 检查 GeminiProvider 是否有有效配置
func (p GeminiProvider) HasValidConfig() bool {
	return p.BaseURL != ""
}

// ============================================================================
// 请求处理公共函数
// ============================================================================

// RequestContext 封装请求上下文信息
type RequestContext struct {
	BodyBytes      []byte            // 原始请求体
	IsStream       bool              // 是否流式请求
	RequestedModel string            // 请求的模型名
	Query          map[string]string // URL 查询参数
	ClientHeaders  map[string]string // 客户端请求头
}

// ReadRequestBody 读取并解析请求体
// 返回 RequestContext 和错误信息
func ReadRequestBody(c *gin.Context) (*RequestContext, error) {
	var bodyBytes []byte
	if c.Request.Body != nil {
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		bodyBytes = data
		// 重置 Body 以便后续使用
		c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	return &RequestContext{
		BodyBytes:      bodyBytes,
		IsStream:       gjson.GetBytes(bodyBytes, "stream").Bool(),
		RequestedModel: gjson.GetBytes(bodyBytes, "model").String(),
		Query:          flattenQuery(c.Request.URL.Query()),
		ClientHeaders:  cloneHeaders(c.Request.Header),
	}, nil
}

// ============================================================================
// Provider 过滤与分组
// ============================================================================

// FilterResult 过滤结果
type FilterResult[T ProviderLike] struct {
	Active       []T // 可用的 providers
	SkippedCount int // 被跳过的数量
}

// FilterProviders 过滤 Provider 列表
// 参数:
//   - providers: 原始 Provider 列表
//   - kind: 平台类型 (claude/codex/gemini)
//   - requestedModel: 请求的模型名（可为空）
//   - blacklistChecker: 黑名单检查函数
//   - modelChecker: 模型支持检查函数（可为 nil）
//   - configValidator: 配置验证函数（可为 nil）
func FilterProviders(
	providers []Provider,
	kind string,
	requestedModel string,
	blacklistChecker func(kind, name string) (bool, time.Time),
	modelChecker func(p *Provider, model string) bool,
	configValidator func(p *Provider) []string,
) FilterResult[Provider] {
	result := FilterResult[Provider]{
		Active: make([]Provider, 0, len(providers)),
	}

	for _, provider := range providers {
		// 基础过滤：启用状态和配置有效性
		if !provider.IsEnabled() || !provider.HasValidConfig() {
			continue
		}

		// 配置验证
		if configValidator != nil {
			if errs := configValidator(&provider); len(errs) > 0 {
				fmt.Printf("[WARN] Provider %s 配置验证失败，已自动跳过: %v\n", provider.Name, errs)
				result.SkippedCount++
				continue
			}
		}

		// 模型支持检查
		if modelChecker != nil && requestedModel != "" {
			if !modelChecker(&provider, requestedModel) {
				fmt.Printf("[INFO] Provider %s 不支持模型 %s，已跳过\n", provider.Name, requestedModel)
				result.SkippedCount++
				continue
			}
		}

		// 黑名单检查
		if blacklistChecker != nil {
			if isBlacklisted, until := blacklistChecker(kind, provider.Name); isBlacklisted {
				fmt.Printf("⛔ Provider %s 已拉黑，过期时间: %v\n", provider.Name, until.Format("15:04:05"))
				result.SkippedCount++
				continue
			}
		}

		result.Active = append(result.Active, provider)
	}

	return result
}

// FilterGeminiProviders 过滤 GeminiProvider 列表
func FilterGeminiProviders(
	providers []GeminiProvider,
	blacklistChecker func(kind, name string) (bool, time.Time),
) FilterResult[GeminiProvider] {
	result := FilterResult[GeminiProvider]{
		Active: make([]GeminiProvider, 0, len(providers)),
	}

	for _, provider := range providers {
		// 基础过滤
		if !provider.IsEnabled() || !provider.HasValidConfig() {
			continue
		}

		// 黑名单检查
		if blacklistChecker != nil {
			if isBlacklisted, until := blacklistChecker("gemini", provider.Name); isBlacklisted {
				fmt.Printf("[Gemini] ⛔ Provider %s 已拉黑，过期时间: %v\n", provider.Name, until.Format("15:04:05"))
				continue
			}
		}

		result.Active = append(result.Active, provider)
	}

	return result
}

// ============================================================================
// Level 分组
// ============================================================================

// LevelGroup 按 Level 分组的结果
type LevelGroup[T ProviderLike] struct {
	Groups       map[int][]T // Level -> Providers 映射
	SortedLevels []int       // 排序后的 Level 列表
}

// GroupByLevel 将 Provider 列表按 Level 分组并排序
func GroupByLevel[T ProviderLike](providers []T) LevelGroup[T] {
	groups := make(map[int][]T)

	for _, p := range providers {
		level := p.GetLevel()
		groups[level] = append(groups[level], p)
	}

	// 提取并排序 levels
	levels := make([]int, 0, len(groups))
	for level := range groups {
		levels = append(levels, level)
	}
	sort.Ints(levels)

	return LevelGroup[T]{
		Groups:       groups,
		SortedLevels: levels,
	}
}

// ============================================================================
// 泛型轮询算法
// ============================================================================

// RoundRobinState 轮询状态管理器
type RoundRobinState struct {
	mu        sync.Mutex
	lastStart map[string]string // key: "platform:level" -> value: 上次起始 Provider Name
}

// NewRoundRobinState 创建轮询状态管理器
func NewRoundRobinState() *RoundRobinState {
	return &RoundRobinState{
		lastStart: make(map[string]string),
	}
}

// Reorder 对 providers 进行轮询排序（泛型版本）
// 算法：将上次起始的 provider 移到末尾，实现负载均衡
// 参数:
//   - platform: 平台标识 (claude/codex/gemini/custom:xxx)
//   - level: 当前 Level
//   - providers: 同 Level 的 providers 列表
//   - getName: 获取 provider 名称的函数
//
// 返回：重新排序后的 providers 列表（新切片，不修改原切片）
func Reorder[T any](
	rrs *RoundRobinState,
	platform string,
	level int,
	providers []T,
	getName func(T) string,
) []T {
	if len(providers) <= 1 {
		return providers
	}

	key := fmt.Sprintf("%s:%d", platform, level)

	rrs.mu.Lock()
	defer rrs.mu.Unlock()

	lastStart := rrs.lastStart[key]

	// 记录本次起始 provider 名称
	rrs.lastStart[key] = getName(providers[0])

	// 如果没有历史记录，返回原顺序
	if lastStart == "" {
		return providers
	}

	// 查找上次起始 provider 在当前列表中的位置
	lastIdx := -1
	for i, p := range providers {
		if getName(p) == lastStart {
			lastIdx = i
			break
		}
	}

	// 上次起始 provider 不在当前列表，返回原顺序
	if lastIdx == -1 {
		return providers
	}

	// 构建轮询顺序：从 lastIdx+1 开始，环形遍历
	result := make([]T, len(providers))
	for i := 0; i < len(providers); i++ {
		idx := (lastIdx + 1 + i) % len(providers)
		result[i] = providers[idx]
	}

	// 更新本次起始 provider 名称
	rrs.lastStart[key] = getName(result[0])

	return result
}

// ============================================================================
// 重试配置
// ============================================================================

// RetryContext 重试上下文
type RetryContext struct {
	MaxRetryPerProvider int           // 每个 Provider 最大重试次数
	RetryWaitDuration   time.Duration // 重试等待时间
	TotalAttempts       int           // 总尝试次数
	LastError           error         // 最后一次错误
	LastProvider        string        // 最后尝试的 Provider
	LastDuration        time.Duration // 最后一次耗时
}

// NewRetryContext 创建重试上下文
func NewRetryContext(failureThreshold int, retryWaitSeconds int) *RetryContext {
	return &RetryContext{
		MaxRetryPerProvider: failureThreshold,
		RetryWaitDuration:   time.Duration(retryWaitSeconds) * time.Second,
	}
}

// RecordAttempt 记录一次尝试
func (rc *RetryContext) RecordAttempt(provider string, duration time.Duration, err error) {
	rc.TotalAttempts++
	rc.LastProvider = provider
	rc.LastDuration = duration
	if err != nil {
		rc.LastError = err
	}
}

// ============================================================================
// 日志记录公共函数
// ============================================================================

// WriteRequestLog 写入请求日志到数据库
func WriteRequestLog(requestLog *ReqeustLog) {
	if GlobalDBQueueLogs == nil {
		fmt.Printf("⚠️  写入 request_log 失败: 队列未初始化\n")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := GlobalDBQueueLogs.ExecBatchCtx(ctx, `
		INSERT INTO request_log (
			platform, model, provider, http_code,
			input_tokens, output_tokens, cache_create_tokens, cache_read_tokens,
			reasoning_tokens, is_stream, duration_sec
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		requestLog.Platform,
		requestLog.Model,
		requestLog.Provider,
		requestLog.HttpCode,
		requestLog.InputTokens,
		requestLog.OutputTokens,
		requestLog.CacheCreateTokens,
		requestLog.CacheReadTokens,
		requestLog.ReasoningTokens,
		boolToInt(requestLog.IsStream),
		requestLog.DurationSec,
	)

	if err != nil {
		fmt.Printf("写入 request_log 失败: %v\n", err)
	}
}

// ============================================================================
// 错误响应构建
// ============================================================================

// BuildFailureResponse 构建失败响应
func BuildFailureResponse(
	totalAttempts int,
	lastProvider string,
	lastError error,
	mode string,
) gin.H {
	errorMsg := "未知错误"
	if lastError != nil {
		errorMsg = lastError.Error()
	}

	response := gin.H{
		"error":         fmt.Sprintf("所有 Provider 都失败，最后尝试: %s - %s", lastProvider, errorMsg),
		"lastProvider":  lastProvider,
		"totalAttempts": totalAttempts,
	}

	if mode == "blacklist" {
		response["mode"] = "blacklist_retry"
		response["hint"] = "拉黑模式已开启，同 Provider 重试到拉黑再切换。如需立即降级请关闭拉黑功能"
	}

	return response
}

// ============================================================================
// 常量定义
// ============================================================================

const (
	// DefaultProviderLevel 默认 Provider 级别
	DefaultProviderLevel = 1

	// DefaultListenAddr 默认监听地址
	DefaultListenAddr = "127.0.0.1:18100"

	// DefaultRequestTimeout 默认请求超时时间
	DefaultRequestTimeout = 32 * time.Hour

	// DefaultGeminiTimeout Gemini 请求超时时间
	DefaultGeminiTimeout = 300 * time.Second

	// DefaultModelsTimeout /v1/models 请求超时时间
	DefaultModelsTimeout = 30 * time.Second
)

// ============================================================================
// Tool Use 完整性修复
// ============================================================================

// FixIncompleteToolUse 检查并修复消息历史中未完成的 tool_use
// 问题场景：当中转站切换时，可能存在 assistant 发出 tool_use 但没有对应 tool_result 的情况
// Claude API 要求每个 tool_use 必须紧跟对应的 tool_result，否则会报错：
// "tool_use ids were found without tool_result blocks immediately after"
//
// 修复策略：检查最后一条 assistant 消息是否包含 tool_use，
// 如果有且没有对应的 tool_result，则补充一个包含错误信息的 tool_result
//
// 参数：
//   - bodyBytes: 原始请求体 (JSON)
//
// 返回：
//   - 修复后的请求体 (如果需要修复) 或原始请求体
//   - 是否进行了修复
//   - 错误信息 (如果有)
func FixIncompleteToolUse(bodyBytes []byte) ([]byte, bool, error) {
	// 获取 messages 数组
	messages := gjson.GetBytes(bodyBytes, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return bodyBytes, false, nil
	}

	messagesArray := messages.Array()
	if len(messagesArray) == 0 {
		return bodyBytes, false, nil
	}

	// 从后向前查找最后一条 assistant 消息
	var lastAssistantIdx int = -1
	var lastAssistantMsg gjson.Result
	for i := len(messagesArray) - 1; i >= 0; i-- {
		msg := messagesArray[i]
		if msg.Get("role").String() == "assistant" {
			lastAssistantIdx = i
			lastAssistantMsg = msg
			break
		}
	}

	// 没有 assistant 消息，无需修复
	if lastAssistantIdx == -1 {
		return bodyBytes, false, nil
	}

	// 检查这条 assistant 消息是否包含 tool_use
	content := lastAssistantMsg.Get("content")
	if !content.Exists() || !content.IsArray() {
		return bodyBytes, false, nil
	}

	// 收集所有 tool_use 的 ID
	var toolUseIDs []string
	content.ForEach(func(_, item gjson.Result) bool {
		if item.Get("type").String() == "tool_use" {
			id := item.Get("id").String()
			if id != "" {
				toolUseIDs = append(toolUseIDs, id)
			}
		}
		return true
	})

	// 没有 tool_use，无需修复
	if len(toolUseIDs) == 0 {
		return bodyBytes, false, nil
	}

	// 检查 assistant 消息之后是否有对应的 tool_result
	// 正常情况下，assistant 消息后面紧跟一个 user 消息，里面包含 tool_result
	hasToolResult := false
	if lastAssistantIdx+1 < len(messagesArray) {
		nextMsg := messagesArray[lastAssistantIdx+1]
		if nextMsg.Get("role").String() == "user" {
			nextContent := nextMsg.Get("content")
			if nextContent.IsArray() {
				nextContent.ForEach(func(_, item gjson.Result) bool {
					if item.Get("type").String() == "tool_result" {
						hasToolResult = true
						return false // 找到了就停止
					}
					return true
				})
			}
		}
	}

	// 如果已有 tool_result，无需修复
	if hasToolResult {
		return bodyBytes, false, nil
	}

	// 需要修复：构建 tool_result 消息
	fmt.Printf("⚠️  检测到未完成的 tool_use (IDs: %v)，正在补充 tool_result...\n", toolUseIDs)

	// 构建 tool_result 内容数组
	toolResults := make([]map[string]interface{}, 0, len(toolUseIDs))
	for _, id := range toolUseIDs {
		toolResults = append(toolResults, map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": id,
			"content":     "工具调用被中断（中转站切换），请重新执行此操作",
			"is_error":    true,
		})
	}

	// 构建新的 user 消息
	newUserMsg := map[string]interface{}{
		"role":    "user",
		"content": toolResults,
	}

	// 使用 sjson 将新消息追加到 messages 数组末尾
	newIdx := len(messagesArray)
	modified, err := sjson.SetBytes(bodyBytes, fmt.Sprintf("messages.%d", newIdx), newUserMsg)
	if err != nil {
		return bodyBytes, false, fmt.Errorf("补充 tool_result 失败: %w", err)
	}

	fmt.Printf("✅ 已补充 %d 个 tool_result，消息历史已修复\n", len(toolUseIDs))
	return modified, true, nil
}
