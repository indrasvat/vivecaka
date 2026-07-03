package usecase

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// GetAttentionInbox builds the cross-repo PR inbox from ranked source data.
type GetAttentionInbox struct {
	reader domain.InboxReader
}

// NewGetAttentionInbox creates the attention inbox use case.
func NewGetAttentionInbox(reader domain.InboxReader) *GetAttentionInbox {
	return &GetAttentionInbox{reader: reader}
}

// Execute fetches and ranks the attention inbox.
func (uc *GetAttentionInbox) Execute(ctx context.Context, query domain.InboxQuery) (*domain.InboxResult, error) {
	if uc == nil || uc.reader == nil {
		return &domain.InboxResult{}, nil
	}
	result, err := uc.reader.GetInbox(ctx, query)
	if err != nil {
		return &domain.InboxResult{}, err
	}
	if result == nil {
		result = &domain.InboxResult{}
	}
	if query.RankProfile == "" {
		query.RankProfile = domain.InboxRankBalanced
	}
	if query.StaleDays <= 0 {
		query.StaleDays = 7
	}
	RankInboxItems(result.Items, query.RankProfile, query.Username, query.StaleDays)
	if query.Limit > 0 && len(result.Items) > query.Limit {
		result.Items = result.Items[:query.Limit]
	}
	return result, nil
}

// RankInboxItems scores items in-place and performs a stable highest-signal sort.
func RankInboxItems(items []domain.InboxItem, profile domain.InboxRankProfile, username string, staleDays int) {
	for i := range items {
		score, reason := rankInboxItem(items[i], profile, username, staleDays)
		items[i].Score = score
		items[i].Reason = reason
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if items[i].Repo.String() != items[j].Repo.String() {
			return items[i].Repo.String() < items[j].Repo.String()
		}
		return items[i].Number < items[j].Number
	})
}

func rankInboxItem(item domain.InboxItem, profile domain.InboxRankProfile, username string, staleDays int) (int, string) {
	score := 0
	reason := "watch"

	if hasInboxSource(item, domain.InboxSourceAttention) {
		score += 1000
		reason = "review"
	}
	if hasInboxSource(item, domain.InboxSourceAssigned) {
		score += 720
		if reason == "watch" {
			reason = "assigned"
		}
	}
	if hasInboxSource(item, domain.InboxSourceFavorite) {
		score += 260
		if profile == domain.InboxRankFavorites {
			score += 350
		}
	}
	if hasInboxSource(item, domain.InboxSourceHome) {
		score += 220
	}
	if hasInboxSource(item, domain.InboxSourceOwned) {
		score += 130
	}

	if strings.EqualFold(item.Author, username) && username != "" {
		score += 35
	}
	if isBotLogin(item.Author) {
		score -= 45
	}
	if item.Draft {
		score -= 60
		if reason == "watch" {
			reason = "draft"
		}
	}

	switch item.CI {
	case domain.CIFail:
		score += 180
		if reason != "review" {
			reason = "ci"
		}
	case domain.CIPending:
		score += 75
	case domain.CIPass:
		if item.Review.State == domain.ReviewApproved {
			score += 45
			if profile == domain.InboxRankMergeReady {
				score += 220
				reason = "ready"
			}
		}
	}

	if item.Review.State == domain.ReviewPending && !strings.EqualFold(item.Author, username) {
		score += 500
		reason = "review"
	}
	if item.Review.State == domain.ReviewChangesRequested {
		score += 120
		if reason == "watch" {
			reason = "review"
		}
	}

	if staleDays > 0 && !item.UpdatedAt.IsZero() {
		threshold := time.Now().Add(-time.Duration(staleDays) * 24 * time.Hour)
		if item.UpdatedAt.Before(threshold) {
			score += 60
			if profile == domain.InboxRankStale {
				score += 220
			}
			if reason == "watch" {
				reason = "stale"
			}
		}
	}

	return score, reason
}

func hasInboxSource(item domain.InboxItem, source domain.InboxSource) bool {
	for _, s := range item.Sources {
		if s == source {
			return true
		}
	}
	return false
}

func isBotLogin(login string) bool {
	lower := strings.ToLower(login)
	return strings.HasSuffix(lower, "[bot]") || strings.Contains(lower, "-bot")
}
