package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/adapter/ghcli"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/views"
	"github.com/indrasvat/vivecaka/internal/usecase"
)

// Default timeout for gh CLI operations.
const ghTimeout = 15 * time.Second

// detectRepoCmd detects the current repo from git remote.
func detectRepoCmd() tea.Cmd {
	return func() tea.Msg {
		repo, err := ghcli.DetectRepo(context.Background())
		return views.RepoDetectedMsg{Repo: repo, Err: err}
	}
}

// detectUserCmd detects the current GitHub user via gh CLI.
func detectUserCmd() tea.Cmd {
	return func() tea.Msg {
		username, err := ghcli.DetectUser(context.Background())
		return views.UserDetectedMsg{Username: username, Err: err}
	}
}

// loadPRsCmd fetches PRs for the given repo.
func loadPRsCmd(uc *usecase.ListPRs, repo domain.RepoRef, opts domain.ListOpts) tea.Cmd {
	return func() tea.Msg {
		prs, err := uc.Execute(context.Background(), repo, opts)
		return views.PRsLoadedMsg{PRs: prs, Err: err}
	}
}

// loadMorePRsCmd fetches additional PRs for pagination.
func loadMorePRsCmd(uc *usecase.ListPRs, repo domain.RepoRef, opts domain.ListOpts, page int) tea.Cmd {
	return func() tea.Msg {
		prs, err := uc.Execute(context.Background(), repo, opts)
		// Determine if there are more pages: if we got fewer than PerPage, no more
		hasMore := len(prs) >= opts.PerPage
		return views.MorePRsLoadedMsg{PRs: prs, Page: page, HasMore: hasMore, Err: err}
	}
}

// loadPRDetailCmd fetches full PR detail with timeout.
func loadPRDetailCmd(uc *usecase.GetPRDetail, repo domain.RepoRef, number int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
		defer cancel()

		detail, err := uc.Execute(ctx, repo, number)
		return views.PRDetailLoadedMsg{Detail: detail, Err: err}
	}
}

// loadDiffCmd fetches the diff for a PR.
func loadDiffCmd(reader domain.PRReader, repo domain.RepoRef, number int) tea.Cmd {
	return func() tea.Msg {
		diff, err := reader.GetDiff(context.Background(), repo, number)
		return views.DiffLoadedMsg{Diff: diff, Err: err}
	}
}

// submitReviewCmd submits a review for a PR.
func submitReviewCmd(uc *usecase.ReviewPR, repo domain.RepoRef, number int, review domain.Review) tea.Cmd {
	return func() tea.Msg {
		err := uc.Execute(context.Background(), repo, number, review)
		return views.ReviewSubmittedMsg{Err: err}
	}
}

// checkoutPRCmd checks out a PR branch.
func checkoutPRCmd(uc *usecase.CheckoutPR, repo domain.RepoRef, number int) tea.Cmd {
	return func() tea.Msg {
		branch, err := uc.Execute(context.Background(), repo, number)
		return views.CheckoutDoneMsg{Branch: branch, Err: err}
	}
}

// resolveThreadCmd resolves a review comment thread.
func resolveThreadCmd(uc *usecase.ResolveThread, repo domain.RepoRef, threadID string) tea.Cmd {
	return func() tea.Msg {
		err := uc.Execute(context.Background(), repo, threadID)
		return resolveThreadDoneMsg{ThreadID: threadID, Err: err}
	}
}
