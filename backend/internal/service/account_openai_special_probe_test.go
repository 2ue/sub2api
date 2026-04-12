package service

import (
	"testing"
	"time"
)

func TestAccount_IsOpenAISpecial429ProbeCandidate(t *testing.T) {
	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)

	t.Run("weekly limited but not five hour limited", func(t *testing.T) {
		account := &Account{
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Extra: map[string]any{
				"codex_7d_used_percent": 100.0,
				"codex_7d_reset_at":     now.Add(2 * time.Hour).Format(time.RFC3339),
				"codex_5h_used_percent": 42.0,
				"codex_5h_reset_at":     now.Add(30 * time.Minute).Format(time.RFC3339),
			},
		}

		if !account.IsOpenAISpecial429ProbeCandidate(now) {
			t.Fatal("expected account to be probe candidate")
		}
	})

	t.Run("five hour limited should be excluded", func(t *testing.T) {
		account := &Account{
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Extra: map[string]any{
				"codex_7d_used_percent": 100.0,
				"codex_7d_reset_at":     now.Add(2 * time.Hour).Format(time.RFC3339),
				"codex_5h_used_percent": 100.0,
				"codex_5h_reset_at":     now.Add(30 * time.Minute).Format(time.RFC3339),
			},
		}

		if account.IsOpenAISpecial429ProbeCandidate(now) {
			t.Fatal("expected five-hour-limited account to be excluded")
		}
	})

	t.Run("special switch already enabled should be excluded", func(t *testing.T) {
		account := &Account{
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Extra: map[string]any{
				"openai_oauth_special_rate_limit_enabled": true,
				"codex_7d_used_percent":                   100.0,
				"codex_7d_reset_at":                       now.Add(2 * time.Hour).Format(time.RFC3339),
			},
		}

		if account.IsOpenAISpecial429ProbeCandidate(now) {
			t.Fatal("expected enabled special-mode account to be excluded")
		}
	})
}

func TestAccount_HasActiveOpenAICodexWeeklyLimit_UsesResetAfterFallback(t *testing.T) {
	now := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"codex_7d_used_percent":        100.0,
			"codex_7d_reset_after_seconds": 180,
			"codex_usage_updated_at":       now.Add(-1 * time.Minute).Format(time.RFC3339),
		},
	}

	if !account.HasActiveOpenAICodexWeeklyLimit(now) {
		t.Fatal("expected reset_after fallback to mark weekly limit active")
	}

	if account.HasActiveOpenAICodexWeeklyLimit(now.Add(3 * time.Minute)) {
		t.Fatal("expected weekly limit to expire after fallback reset time")
	}
}
