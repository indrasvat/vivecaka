package usecase

import (
	"context"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// GetInboxInsight fetches lazy selected-row insight data for one inbox PR.
type GetInboxInsight struct {
	reader domain.InboxInsightReader
}

// NewGetInboxInsight creates a selected-row inbox insight use case.
func NewGetInboxInsight(reader domain.InboxInsightReader) *GetInboxInsight {
	return &GetInboxInsight{reader: reader}
}

// Execute returns compact PR insight data. A nil reader returns an empty insight.
func (uc *GetInboxInsight) Execute(ctx context.Context, query domain.InboxInsightQuery) (*domain.InboxInsight, error) {
	if uc == nil || uc.reader == nil || query.Repo.Owner == "" || query.Number <= 0 {
		return &domain.InboxInsight{Repo: query.Repo, Number: query.Number, LocalState: query.LocalState}, nil
	}
	return uc.reader.GetInboxInsight(ctx, query)
}
