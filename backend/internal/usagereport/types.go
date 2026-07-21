package usagereport

import (
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrForbiddenCrossUser = infraerrors.Forbidden("REPORT_FORBIDDEN", "cross-user access requires admin")
	ErrInvalidDateRange   = infraerrors.BadRequest("REPORT_INVALID_DATE_RANGE", "start_date and end_date are required")
	ErrInvalidUserID      = infraerrors.BadRequest("REPORT_INVALID_USER_ID", "invalid user id")
	ErrUserNotFound       = infraerrors.NotFound("REPORT_USER_NOT_FOUND", "user not found")
)

type CurrentUser struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username,omitempty"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name,omitempty"`
	Deleted     bool   `json:"deleted,omitempty"`
}

type BootstrapResponse struct {
	CurrentUser    CurrentUser `json:"current_user"`
	CanSearchUsers bool        `json:"can_search_users"`
	Timezone       string      `json:"timezone"`
}

type UserOption struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username,omitempty"`
	Role        string `json:"role,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Deleted     bool   `json:"deleted"`
}

type SummaryRequest struct {
	UserID    *int64 `json:"user_id,omitempty"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Timezone  string `json:"timezone,omitempty"`
}

type SummaryQueryInfo struct {
	CurrentUser CurrentUser `json:"current_user"`
	TargetUser  CurrentUser `json:"target_user"`
	StartDate   string      `json:"start_date"`
	EndDate     string      `json:"end_date"`
	Timezone    string      `json:"timezone"`
}

type SummaryTotals struct {
	Requests    int64   `json:"requests"`
	TotalTokens int64   `json:"total_tokens"`
	ActualCost  float64 `json:"actual_cost"`
}

type GroupBreakdownRow struct {
	GroupID     int64   `json:"group_id"`
	GroupName   string  `json:"group_name"`
	Platform    string  `json:"platform"`
	Requests    int64   `json:"requests"`
	TotalTokens int64   `json:"total_tokens"`
	ActualCost  float64 `json:"actual_cost"`
}

type PlatformBreakdownRow struct {
	Platform    string  `json:"platform"`
	Requests    int64   `json:"requests"`
	TotalTokens int64   `json:"total_tokens"`
	ActualCost  float64 `json:"actual_cost"`
}

type SummaryResponse struct {
	Query            SummaryQueryInfo       `json:"query"`
	Totals           SummaryTotals          `json:"totals"`
	GroupBreakdown   []GroupBreakdownRow    `json:"group_breakdown"`
	PlatformBreakdown []PlatformBreakdownRow `json:"platform_breakdown"`
}

func displayName(email, username string) string {
	username = strings.TrimSpace(username)
	if username != "" {
		return username
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return email
}

func toCurrentUser(id int64, email, username, role string, deleted bool) CurrentUser {
	return CurrentUser{
		ID:          id,
		Email:       strings.TrimSpace(email),
		Username:    strings.TrimSpace(username),
		Role:        strings.TrimSpace(role),
		DisplayName: displayName(email, username),
		Deleted:     deleted,
	}
}

func toUserOption(id int64, email, username, role string, deleted bool) UserOption {
	return UserOption{
		ID:          id,
		Email:       strings.TrimSpace(email),
		Username:    strings.TrimSpace(username),
		Role:        strings.TrimSpace(role),
		DisplayName: displayName(email, username),
		Deleted:     deleted,
	}
}

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 30:
		return 30
	default:
		return limit
	}
}

func requireSummaryRange(startDate, endDate string) error {
	if strings.TrimSpace(startDate) == "" || strings.TrimSpace(endDate) == "" {
		return ErrInvalidDateRange
	}
	return nil
}

func validateUserID(userID *int64) error {
	if userID == nil {
		return nil
	}
	if *userID <= 0 {
		return ErrInvalidUserID
	}
	return nil
}

func summarizeRequestRange(startDate, endDate, timezone string) string {
	return fmt.Sprintf("%s..%s@%s", strings.TrimSpace(startDate), strings.TrimSpace(endDate), strings.TrimSpace(timezone))
}

