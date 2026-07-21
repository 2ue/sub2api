package usagereport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"golang.org/x/sync/errgroup"
)

type reportUserRepository interface {
	GetByID(ctx context.Context, id int64) (*service.User, error)
	GetByIDIncludeDeleted(ctx context.Context, id int64) (*service.User, error)
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error)
}

type reportService struct {
	db       *sql.DB
	userRepo reportUserRepository
}

const groupBreakdownQuery = `
		SELECT
			COALESCE(ul.group_id, 0)::bigint AS group_id,
			COALESCE(NULLIF(g.name, ''), 'Ungrouped') AS group_name,
			COALESCE(NULLIF(g.platform, ''), NULLIF(a.platform, ''), 'unknown') AS platform,
			COUNT(*)::bigint AS requests,
			COALESCE(SUM(
				COALESCE(ul.input_tokens, 0)
				+ COALESCE(ul.output_tokens, 0)
				+ COALESCE(ul.cache_creation_tokens, 0)
				+ COALESCE(ul.cache_read_tokens, 0)
			), 0)::bigint AS total_tokens,
			COALESCE(SUM(ul.actual_cost), 0)::float8 AS actual_cost
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		LEFT JOIN accounts a ON a.id = ul.account_id
		WHERE ul.user_id = $1
		  AND ul.created_at >= $2
		  AND ul.created_at < $3
		  AND ul.actual_cost > 0
		GROUP BY COALESCE(ul.group_id, 0), g.name, g.platform, a.platform
		ORDER BY actual_cost DESC, requests DESC, group_name ASC
	`

const platformBreakdownQuery = `
		SELECT
			COALESCE(NULLIF(g.platform, ''), NULLIF(a.platform, ''), 'unknown') AS platform,
			COUNT(*)::bigint AS requests,
			COALESCE(SUM(
				COALESCE(ul.input_tokens, 0)
				+ COALESCE(ul.output_tokens, 0)
				+ COALESCE(ul.cache_creation_tokens, 0)
				+ COALESCE(ul.cache_read_tokens, 0)
			), 0)::bigint AS total_tokens,
			COALESCE(SUM(ul.actual_cost), 0)::float8 AS actual_cost
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		LEFT JOIN accounts a ON a.id = ul.account_id
		WHERE ul.user_id = $1
		  AND ul.created_at >= $2
		  AND ul.created_at < $3
		  AND ul.actual_cost > 0
		GROUP BY COALESCE(NULLIF(g.platform, ''), NULLIF(a.platform, ''), 'unknown')
		ORDER BY actual_cost DESC, requests DESC, platform ASC
	`

func NewService(db *sql.DB, userRepo reportUserRepository) *reportService {
	return &reportService{db: db, userRepo: userRepo}
}

func (s *reportService) Bootstrap(ctx context.Context, current *service.User, canSearchUsers bool) BootstrapResponse {
	return BootstrapResponse{
		CurrentUser:    toCurrentUser(current.ID, current.Email, current.Username, current.Role, current.DeletedAt != nil),
		CanSearchUsers: canSearchUsers,
		Timezone:       timezone.Name(),
	}
}

func (s *reportService) SearchUsers(ctx context.Context, actorRole, query string, limit int) ([]UserOption, error) {
	if strings.TrimSpace(actorRole) != service.RoleAdmin {
		return nil, ErrForbiddenCrossUser
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return []UserOption{}, nil
	}
	limit = normalizeLimit(limit)
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}

	users, _, err := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{
		Page:     1,
		PageSize: limit,
		SortBy:   "email",
		SortOrder: "asc",
	}, service.UserListFilters{
		Search:          query,
		IncludeDeleted:  true,
		IncludeSubscriptions: nil,
	})
	if err != nil {
		return nil, err
	}

	out := make([]UserOption, 0, len(users))
	for i := range users {
		out = append(out, toUserOption(users[i].ID, users[i].Email, users[i].Username, users[i].Role, users[i].DeletedAt != nil))
	}
	return out, nil
}

func (s *reportService) QuerySummary(ctx context.Context, actor *service.User, actorRole string, req SummaryRequest) (*SummaryResponse, error) {
	if err := requireSummaryRange(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}
	if err := validateUserID(req.UserID); err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, fmt.Errorf("current user is required")
	}
	if s.db == nil {
		return nil, fmt.Errorf("database is not configured")
	}

	target, err := s.resolveTargetUser(ctx, actor, actorRole, req.UserID)
	if err != nil {
		return nil, err
	}

	start, end, tzName, err := parseRange(req.StartDate, req.EndDate, req.Timezone)
	if err != nil {
		return nil, err
	}

	groupRows := make([]GroupBreakdownRow, 0)
	platformRows := make([]PlatformBreakdownRow, 0)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		rows, err := s.queryGroupBreakdown(gctx, target.ID, start, end)
		if err != nil {
			return err
		}
		groupRows = rows
		return nil
	})
	g.Go(func() error {
		rows, err := s.queryPlatformBreakdown(gctx, target.ID, start, end)
		if err != nil {
			return err
		}
		platformRows = rows
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	totals := SummaryTotals{}
	for _, row := range groupRows {
		totals.Requests += row.Requests
		totals.TotalTokens += row.TotalTokens
		totals.ActualCost += row.ActualCost
	}

	return &SummaryResponse{
		Query: SummaryQueryInfo{
			CurrentUser: toCurrentUser(actor.ID, actor.Email, actor.Username, actor.Role, actor.DeletedAt != nil),
			TargetUser:  toCurrentUser(target.ID, target.Email, target.Username, target.Role, target.DeletedAt != nil),
			StartDate:   strings.TrimSpace(req.StartDate),
			EndDate:     strings.TrimSpace(req.EndDate),
			Timezone:    tzName,
		},
		Totals:            totals,
		GroupBreakdown:    groupRows,
		PlatformBreakdown: platformRows,
	}, nil
}

func (s *reportService) resolveTargetUser(ctx context.Context, actor *service.User, actorRole string, userID *int64) (*service.User, error) {
	if userID == nil || *userID == actor.ID {
		return actor, nil
	}
	if strings.TrimSpace(actorRole) != service.RoleAdmin {
		return nil, ErrForbiddenCrossUser
	}
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}
	target, err := s.userRepo.GetByIDIncludeDeleted(ctx, *userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return target, nil
}

func parseRange(startDate, endDate, tzName string) (time.Time, time.Time, string, error) {
	loc := timezone.Location()
	if tzName = strings.TrimSpace(tzName); tzName != "" {
		if parsed, err := time.LoadLocation(tzName); err == nil {
			loc = parsed
		}
	}

	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), loc)
	if err != nil {
		return time.Time{}, time.Time{}, "", ErrInvalidDateRange
	}
	end, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), loc)
	if err != nil {
		return time.Time{}, time.Time{}, "", ErrInvalidDateRange
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, "", ErrInvalidDateRange
	}

	// Use a half-open interval [start, next-day-after-end).
	endExclusive := time.Date(end.Year(), end.Month(), end.Day()+1, 0, 0, 0, 0, loc)
	return start.UTC(), endExclusive.UTC(), loc.String(), nil
}

func (s *reportService) queryGroupBreakdown(ctx context.Context, userID int64, start, end time.Time) ([]GroupBreakdownRow, error) {
	if userID <= 0 {
		return []GroupBreakdownRow{}, nil
	}
	rows, err := s.db.QueryContext(ctx, groupBreakdownQuery, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]GroupBreakdownRow, 0)
	for rows.Next() {
		var row GroupBreakdownRow
		if err := rows.Scan(&row.GroupID, &row.GroupName, &row.Platform, &row.Requests, &row.TotalTokens, &row.ActualCost); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *reportService) queryPlatformBreakdown(ctx context.Context, userID int64, start, end time.Time) ([]PlatformBreakdownRow, error) {
	if userID <= 0 {
		return []PlatformBreakdownRow{}, nil
	}
	rows, err := s.db.QueryContext(ctx, platformBreakdownQuery, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]PlatformBreakdownRow, 0)
	for rows.Next() {
		var row PlatformBreakdownRow
		if err := rows.Scan(&row.Platform, &row.Requests, &row.TotalTokens, &row.ActualCost); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
