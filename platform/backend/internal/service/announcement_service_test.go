package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type announcementRepoStub struct {
	item *Announcement
}

func (s *announcementRepoStub) Create(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (s *announcementRepoStub) GetByID(_ context.Context, _ int64) (*Announcement, error) {
	if s.item == nil {
		return nil, ErrAnnouncementNotFound
	}
	return s.item, nil
}

func (s *announcementRepoStub) Update(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (*announcementRepoStub) Delete(context.Context, int64) error {
	return nil
}

func (*announcementRepoStub) List(context.Context, pagination.PaginationParams, AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (*announcementRepoStub) ListActive(context.Context, time.Time) ([]Announcement, error) {
	return nil, nil
}

func TestAnnouncementServiceCreateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil)
	now := time.Unix(1776790020, 0)

	_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "公告",
		Content:    "内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModePopup,
		StartsAt:   &now,
		EndsAt:     &now,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceUpdateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         1,
			Title:      "公告",
			Content:    "内容",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
		},
	}
	svc := NewAnnouncementService(repo, nil, nil, nil)
	now := time.Unix(1776790020, 0)
	startsAt := &now
	endsAt := &now

	_, err := svc.Update(context.Background(), 1, &UpdateAnnouncementInput{
		StartsAt: &startsAt,
		EndsAt:   &endsAt,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceDashboardPosterFieldsAndLinkValidation(t *testing.T) {
	t.Run("create accepts empty, relative path, and https url", func(t *testing.T) {
		validLinks := []string{
			"",
			"  ",
			"/purchase?amount=100",
			"/model-plaza",
			"https://gabugu.com/docs/launch?ref=poster",
		}
		for _, link := range validLinks {
			repo := &announcementRepoStub{}
			svc := NewAnnouncementService(repo, nil, nil, nil)
			created, err := svc.Create(context.Background(), &CreateAnnouncementInput{
				Title:           "活动海报",
				Content:         "正文",
				Status:          AnnouncementStatusActive,
				NotifyMode:      AnnouncementNotifyModeSilent,
				ShowOnDashboard: true,
				DashboardLink:   link,
			})
			require.NoError(t, err, "link=%q", link)
			require.True(t, created.ShowOnDashboard)
			if link == "  " {
				require.Equal(t, "", created.DashboardLink)
			} else {
				require.Equal(t, link, created.DashboardLink)
			}
		}
	})

	t.Run("create rejects unsafe or malformed dashboard links", func(t *testing.T) {
		longLink := "/purchase?" + string(make([]byte, 500))
		invalidLinks := []string{
			"javascript:alert(1)",
			"data:text/html,<script>alert(1)</script>",
			"http://gabugu.com/promo",
			"//evil.com/phish",
			"/\\evil.com",
			"https://user:pass@gabugu.com/promo",
			"https://gabugu.com/has space",
			"https://gabugu.com/ctrl\nchar",
			"https://gabugu.com/c1\u0085char",
			"relative-without-slash",
			longLink,
		}
		for _, link := range invalidLinks {
			repo := &announcementRepoStub{}
			svc := NewAnnouncementService(repo, nil, nil, nil)
			_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
				Title:           "活动海报",
				Content:         "正文",
				Status:          AnnouncementStatusDraft,
				NotifyMode:      AnnouncementNotifyModeSilent,
				ShowOnDashboard: true,
				DashboardLink:   link,
			})
			require.ErrorIs(t, err, ErrAnnouncementInvalidDashboardLink, "link=%q", link)
		}
	})

	t.Run("update toggles show_on_dashboard and validates dashboard_link", func(t *testing.T) {
		repo := &announcementRepoStub{
			item: &Announcement{
				ID:              7,
				Title:           "公告",
				Content:         "内容",
				Status:          AnnouncementStatusActive,
				NotifyMode:      AnnouncementNotifyModeSilent,
				ShowOnDashboard: false,
				DashboardLink:   "",
			},
		}
		svc := NewAnnouncementService(repo, nil, nil, nil)

		show := true
		goodLink := " /studio "
		updated, err := svc.Update(context.Background(), 7, &UpdateAnnouncementInput{
			ShowOnDashboard: &show,
			DashboardLink:   &goodLink,
		})
		require.NoError(t, err)
		require.True(t, updated.ShowOnDashboard)
		require.Equal(t, "/studio", updated.DashboardLink)

		badLink := "javascript:void(0)"
		_, err = svc.Update(context.Background(), 7, &UpdateAnnouncementInput{
			DashboardLink: &badLink,
		})
		require.ErrorIs(t, err, ErrAnnouncementInvalidDashboardLink)
	})
}
