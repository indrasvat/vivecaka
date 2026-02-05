package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/adapter/ghcli"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/tui/views"
	"github.com/indrasvat/vivecaka/internal/usecase"
)

// detectRepoCmd detects the current repo from git remote.
func detectRepoCmd() tea.Cmd {
	return func() tea.Msg {
		repo, err := ghcli.DetectRepo(context.Background())
		return views.RepoDetectedMsg{Repo: repo, Err: err}
	}
}

// loadPRsCmd fetches PRs for the given repo.
func loadPRsCmd(uc *usecase.ListPRs, repo domain.RepoRef, opts domain.ListOpts) tea.Cmd {
	return func() tea.Msg {
		prs, err := uc.Execute(context.Background(), repo, opts)
		return views.PRsLoadedMsg{PRs: prs, Err: err}
	}
}

// loadPRDetailCmd fetches full PR detail.
func loadPRDetailCmd(uc *usecase.GetPRDetail, repo domain.RepoRef, number int) tea.Cmd {
	return func() tea.Msg {
		detail, err := uc.Execute(context.Background(), repo, number)
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
