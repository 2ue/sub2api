package service

import (
	"context"
	"time"
)

// TempUnschedState 临时不可调度状态
type TempUnschedState struct {
	UntilUnix       int64  `json:"until_unix"`        // 解除时间（Unix 时间戳）
	TriggeredAtUnix int64  `json:"triggered_at_unix"` // 触发时间（Unix 时间戳）
	StatusCode      int    `json:"status_code"`       // 触发的错误码
	MatchedKeyword  string `json:"matched_keyword"`   // 匹配的关键词
	RuleIndex       int    `json:"rule_index"`        // 触发的规则索引
	ErrorMessage    string `json:"error_message"`     // 错误消息
}

// TempUnschedCache 临时不可调度缓存接口
type TempUnschedCache interface {
	SetTempUnsched(ctx context.Context, accountID int64, state *TempUnschedState) error
	GetTempUnsched(ctx context.Context, accountID int64) (*TempUnschedState, error)
	DeleteTempUnsched(ctx context.Context, accountID int64) error
}

// TimeoutCounterCache 超时计数器缓存接口
type TimeoutCounterCache interface {
	// IncrementTimeoutCount 增加账户的超时计数，返回当前计数值
	// windowMinutes 是计数窗口时间（分钟），超过此时间计数器会自动重置
	IncrementTimeoutCount(ctx context.Context, accountID int64, windowMinutes int) (int64, error)
	// GetTimeoutCount 获取账户当前的超时计数
	GetTimeoutCount(ctx context.Context, accountID int64) (int64, error)
	// ResetTimeoutCount 重置账户的超时计数
	ResetTimeoutCount(ctx context.Context, accountID int64) error
	// GetTimeoutCountTTL 获取计数器剩余过期时间
	GetTimeoutCountTTL(ctx context.Context, accountID int64) (time.Duration, error)
}

// OpenAI429CounterCache 追踪特殊 OpenAI OAuth 账号在窗口内的 429 次数。
type OpenAI429CounterCache interface {
	// IncrementOpenAI429Count 原子递增计数并返回当前值。
	// windowSeconds 是计数窗口时间（秒），超过此时间计数器会自动重置。
	IncrementOpenAI429Count(ctx context.Context, accountID int64, windowSeconds int) (int64, error)
	// ResetOpenAI429Count 清零计数器（成功响应或手动恢复状态时调用）。
	ResetOpenAI429Count(ctx context.Context, accountID int64) error
}
