package usagereport

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type stubReportUserRepo struct {
	getByID               func(context.Context, int64) (*service.User, error)
	getByIDIncludeDeleted func(context.Context, int64) (*service.User, error)
	listWithFilters       func(context.Context, pagination.PaginationParams, service.UserListFilters) ([]service.User, *pagination.PaginationResult, error)
}

func (r *stubReportUserRepo) GetByID(ctx context.Context, id int64) (*service.User, error) {
	if r != nil && r.getByID != nil {
		return r.getByID(ctx, id)
	}
	return nil, service.ErrUserNotFound
}

func (r *stubReportUserRepo) GetByIDIncludeDeleted(ctx context.Context, id int64) (*service.User, error) {
	if r != nil && r.getByIDIncludeDeleted != nil {
		return r.getByIDIncludeDeleted(ctx, id)
	}
	return nil, service.ErrUserNotFound
}

func (r *stubReportUserRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	if r != nil && r.listWithFilters != nil {
		return r.listWithFilters(ctx, params, filters)
	}
	return nil, nil, fmt.Errorf("unexpected ListWithFilters call")
}

func newActiveUser(id int64, role string) *service.User {
	return &service.User{
		ID:       id,
		Email:    fmt.Sprintf("user%d@example.com", id),
		Username: fmt.Sprintf("user%d", id),
		Role:     role,
		Status:   service.StatusActive,
	}
}

func newSummaryMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
	})
	return db, mock
}

func expectedUTCDateRange(t *testing.T, startDate, endDate, tzName string) (time.Time, time.Time) {
	t.Helper()

	loc, err := time.LoadLocation(tzName)
	require.NoError(t, err)

	start := time.Date(mustDatePart(t, startDate), mustMonthPart(t, startDate), mustDayPart(t, startDate), 0, 0, 0, 0, loc).UTC()
	end := time.Date(mustDatePart(t, endDate), mustMonthPart(t, endDate), mustDayPart(t, endDate)+1, 0, 0, 0, 0, loc).UTC()
	return start, end
}

func mustDatePart(t *testing.T, date string) int {
	t.Helper()

	parsed, err := time.Parse("2006-01-02", date)
	require.NoError(t, err)
	return parsed.Year()
}

func mustMonthPart(t *testing.T, date string) time.Month {
	t.Helper()

	parsed, err := time.Parse("2006-01-02", date)
	require.NoError(t, err)
	return parsed.Month()
}

func mustDayPart(t *testing.T, date string) int {
	t.Helper()

	parsed, err := time.Parse("2006-01-02", date)
	require.NoError(t, err)
	return parsed.Day()
}

func TestSummaryQuerySQLShape(t *testing.T) {
	require.Contains(t, groupBreakdownQuery, "ul.actual_cost > 0")
	require.Contains(t, platformBreakdownQuery, "ul.actual_cost > 0")
	require.Contains(t, groupBreakdownQuery, "COALESCE(NULLIF(g.platform, ''), NULLIF(a.platform, ''), 'unknown')")
	require.Contains(t, platformBreakdownQuery, "COALESCE(NULLIF(g.platform, ''), NULLIF(a.platform, ''), 'unknown')")
}

func TestParseRange(t *testing.T) {
	t.Run("uses provided timezone", func(t *testing.T) {
		start, end, tz, err := parseRange("2026-07-21", "2026-07-22", "Asia/Tokyo")
		require.NoError(t, err)

		loc, err := time.LoadLocation("Asia/Tokyo")
		require.NoError(t, err)

		require.Equal(t, time.Date(2026, 7, 21, 0, 0, 0, 0, loc).UTC(), start)
		require.Equal(t, time.Date(2026, 7, 23, 0, 0, 0, 0, loc).UTC(), end)
		require.Equal(t, "Asia/Tokyo", tz)
	})

	t.Run("falls back to service timezone when timezone is missing or invalid", func(t *testing.T) {
		start1, end1, tz1, err := parseRange("2026-07-21", "2026-07-22", "")
		require.NoError(t, err)

		start2, end2, tz2, err := parseRange("2026-07-21", "2026-07-22", "not/a-zone")
		require.NoError(t, err)

		require.Equal(t, start1, start2)
		require.Equal(t, end1, end2)
		require.Equal(t, tz1, tz2)
		require.NotEmpty(t, tz1)
	})

	t.Run("rejects reversed dates", func(t *testing.T) {
		_, _, _, err := parseRange("2026-07-22", "2026-07-21", "Asia/Tokyo")
		require.ErrorIs(t, err, ErrInvalidDateRange)
	})
}

func TestSearchUsers(t *testing.T) {
	t.Run("admin search returns minimal records and normalizes limit", func(t *testing.T) {
		deletedAt := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
		repo := &stubReportUserRepo{
			listWithFilters: func(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
				require.Equal(t, 1, params.Page)
				require.Equal(t, 30, params.PageSize)
				require.Equal(t, "email", params.SortBy)
				require.Equal(t, "asc", params.SortOrder)
				require.Equal(t, "ali", filters.Search)
				require.True(t, filters.IncludeDeleted)
				require.Nil(t, filters.IncludeSubscriptions)
				return []service.User{
					{
						ID:        2,
						Email:     "alice@example.com",
						Username:  "alice",
						Role:      service.RoleUser,
						Status:    service.StatusActive,
						DeletedAt: &deletedAt,
					},
				}, &pagination.PaginationResult{Total: 1, Page: 1, PageSize: 30, Pages: 1}, nil
			},
		}

		svc := NewService(nil, repo)
		users, err := svc.SearchUsers(context.Background(), service.RoleAdmin, " ali ", 50)
		require.NoError(t, err)
		require.Len(t, users, 1)
		require.Equal(t, int64(2), users[0].ID)
		require.Equal(t, "alice@example.com", users[0].Email)
		require.Equal(t, "alice", users[0].DisplayName)
		require.True(t, users[0].Deleted)
	})

	t.Run("non-admin callers are rejected before repository access", func(t *testing.T) {
		repo := &stubReportUserRepo{
			listWithFilters: func(context.Context, pagination.PaginationParams, service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
				t.Fatal("ListWithFilters should not be called for non-admin callers")
				return nil, nil, nil
			},
		}

		svc := NewService(nil, repo)
		_, err := svc.SearchUsers(context.Background(), service.RoleUser, "ali", 20)
		require.ErrorIs(t, err, ErrForbiddenCrossUser)
	})
}

func TestQuerySummary(t *testing.T) {
	t.Run("self query returns totals and breakdowns", func(t *testing.T) {
		db, mock := newSummaryMock(t)
		svc := NewService(db, &stubReportUserRepo{})

		current := newActiveUser(11, service.RoleUser)
		req := SummaryRequest{
			StartDate: "2026-07-21",
			EndDate:   "2026-07-22",
			Timezone:  "Asia/Tokyo",
		}

		start, end := expectedUTCDateRange(t, req.StartDate, req.EndDate, req.Timezone)
		mock.ExpectQuery(regexp.QuoteMeta(groupBreakdownQuery)).
			WithArgs(current.ID, start, end).
			WillReturnRows(sqlmock.NewRows([]string{
				"group_id",
				"group_name",
				"platform",
				"requests",
				"total_tokens",
				"actual_cost",
			}).AddRow(int64(7), "Team Alpha", "openai", int64(2), int64(30), 3.50).
				AddRow(int64(0), "Ungrouped", "anthropic", int64(1), int64(12), 1.25))
		mock.ExpectQuery(regexp.QuoteMeta(platformBreakdownQuery)).
			WithArgs(current.ID, start, end).
			WillReturnRows(sqlmock.NewRows([]string{
				"platform",
				"requests",
				"total_tokens",
				"actual_cost",
			}).AddRow("openai", int64(2), int64(30), 3.50).
				AddRow("anthropic", int64(1), int64(12), 1.25))

		summary, err := svc.QuerySummary(context.Background(), current, service.RoleUser, req)
		require.NoError(t, err)
		require.Equal(t, current.ID, summary.Query.CurrentUser.ID)
		require.Equal(t, current.ID, summary.Query.TargetUser.ID)
		require.Equal(t, req.StartDate, summary.Query.StartDate)
		require.Equal(t, req.EndDate, summary.Query.EndDate)
		require.Equal(t, "Asia/Tokyo", summary.Query.Timezone)
		require.Equal(t, int64(3), summary.Totals.Requests)
		require.Equal(t, int64(42), summary.Totals.TotalTokens)
		require.InDelta(t, 4.75, summary.Totals.ActualCost, 0.0001)
		require.Len(t, summary.GroupBreakdown, 2)
		require.Equal(t, "Team Alpha", summary.GroupBreakdown[0].GroupName)
		require.Equal(t, "openai", summary.GroupBreakdown[0].Platform)
		require.Len(t, summary.PlatformBreakdown, 2)
		require.Equal(t, "anthropic", summary.PlatformBreakdown[1].Platform)
	})

	t.Run("admin callers can query another user", func(t *testing.T) {
		db, mock := newSummaryMock(t)
		target := newActiveUser(27, service.RoleUser)
		repo := &stubReportUserRepo{
			getByIDIncludeDeleted: func(ctx context.Context, id int64) (*service.User, error) {
				require.Equal(t, target.ID, id)
				return target, nil
			},
		}
		svc := NewService(db, repo)

		current := newActiveUser(11, service.RoleAdmin)
		targetID := target.ID
		req := SummaryRequest{
			UserID:    &targetID,
			StartDate: "2026-07-21",
			EndDate:   "2026-07-22",
			Timezone:  "Asia/Tokyo",
		}

		start, end := expectedUTCDateRange(t, req.StartDate, req.EndDate, req.Timezone)
		mock.ExpectQuery(regexp.QuoteMeta(groupBreakdownQuery)).
			WithArgs(target.ID, start, end).
			WillReturnRows(sqlmock.NewRows([]string{
				"group_id",
				"group_name",
				"platform",
				"requests",
				"total_tokens",
				"actual_cost",
			}).AddRow(int64(3), "Admin Team", "openai", int64(4), int64(88), 6.25))
		mock.ExpectQuery(regexp.QuoteMeta(platformBreakdownQuery)).
			WithArgs(target.ID, start, end).
			WillReturnRows(sqlmock.NewRows([]string{
				"platform",
				"requests",
				"total_tokens",
				"actual_cost",
			}).AddRow("openai", int64(4), int64(88), 6.25))

		summary, err := svc.QuerySummary(context.Background(), current, service.RoleAdmin, req)
		require.NoError(t, err)
		require.Equal(t, current.ID, summary.Query.CurrentUser.ID)
		require.Equal(t, target.ID, summary.Query.TargetUser.ID)
		require.Equal(t, int64(4), summary.Totals.Requests)
		require.Equal(t, int64(88), summary.Totals.TotalTokens)
		require.InDelta(t, 6.25, summary.Totals.ActualCost, 0.0001)
		require.Len(t, summary.GroupBreakdown, 1)
		require.Len(t, summary.PlatformBreakdown, 1)
	})

	t.Run("missing date range is rejected before database access", func(t *testing.T) {
		svc := NewService(nil, &stubReportUserRepo{})
		_, err := svc.QuerySummary(context.Background(), newActiveUser(11, service.RoleUser), service.RoleUser, SummaryRequest{
			EndDate:  "2026-07-22",
			Timezone: "Asia/Tokyo",
		})
		require.ErrorIs(t, err, ErrInvalidDateRange)
	})

	t.Run("non-admin cross-user query is rejected before target lookup", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)

		repo := &stubReportUserRepo{
			getByIDIncludeDeleted: func(context.Context, int64) (*service.User, error) {
				t.Fatal("GetByIDIncludeDeleted should not be called for non-admin callers")
				return nil, nil
			},
		}
		svc := NewService(db, repo)

		targetID := int64(99)
		_, err = svc.QuerySummary(context.Background(), newActiveUser(11, service.RoleUser), service.RoleUser, SummaryRequest{
			UserID:    &targetID,
			StartDate: "2026-07-21",
			EndDate:   "2026-07-22",
			Timezone:  "Asia/Tokyo",
		})
		require.ErrorIs(t, err, ErrForbiddenCrossUser)
	})
}
