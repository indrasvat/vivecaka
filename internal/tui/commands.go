package tui

import (
	"context"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/indrasvat/vivecaka/internal/adapter/ghcli"
	"github.com/indrasvat/vivecaka/internal/cache"
	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/logging"
	"github.com/indrasvat/vivecaka/internal/tui/views"
	"github.com/indrasvat/vivecaka/internal/usecase"
)

// Default timeout for gh CLI operations.
const ghTimeout = 15 * time.Second

// diffTimeout is longer than ghTimeout because large PRs legitimately take time.
const diffTimeout = 60 * time.Second

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

// loadPRCountCmd fetches the total PR count for a repo.
func loadPRCountCmd(reader domain.PRReader, repo domain.RepoRef, state domain.PRState) tea.Cmd {
	return func() tea.Msg {
		count, err := reader.GetPRCount(context.Background(), repo, state)
		return views.PRCountLoadedMsg{Total: count, Err: err}
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

// loadDiffCmd fetches the diff for a PR with a timeout.
func loadDiffCmd(reader domain.PRReader, repo domain.RepoRef, number int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), diffTimeout)
		defer cancel()

		start := time.Now()
		logging.Log.Debug("loading diff", "pr", number, "repo", repo.String())
		diff, err := reader.GetDiff(ctx, repo, number)
		elapsed := time.Since(start)
		if err != nil {
			logging.Log.Debug("diff load failed", "pr", number, "elapsed", elapsed, "err", err)
		} else {
			files := 0
			if diff != nil {
				files = len(diff.Files)
			}
			logging.Log.Debug("diff loaded", "pr", number, "elapsed", elapsed, "files", files)
		}
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

// detectBranchCmd detects the current git branch.
func detectBranchCmd() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(context.Background(), "git", "rev-parse", "--abbrev-ref", "HEAD")
		out, err := cmd.Output()
		branch := strings.TrimSpace(string(out))
		if err != nil || branch == "" {
			return views.BranchDetectedMsg{Err: err}
		}
		return views.BranchDetectedMsg{Branch: branch}
	}
}

// discoverReposCmd fetches the user's repos via gh repo list.
func discoverReposCmd() tea.Cmd {
	return func() tea.Msg {
		repos, err := ghcli.ListUserRepos(context.Background(), 20)
		return views.ReposDiscoveredMsg{Repos: repos, Err: err}
	}
}

// loadInboxCmd fetches PRs from multiple repos for the inbox.
func loadInboxCmd(uc *usecase.GetInboxPRs, repos []domain.RepoRef) tea.Cmd {
	return func() tea.Msg {
		prs, err := uc.Execute(context.Background(), repos)
		if err != nil {
			return views.InboxPRsLoadedMsg{PRs: nil}
		}
		// Convert usecase.InboxPR to views.InboxPR.
		vPRs := make([]views.InboxPR, len(prs))
		for i, p := range prs {
			vPRs[i] = views.InboxPR{PR: p.PR, Repo: p.Repo}
		}
		return views.InboxPRsLoadedMsg{PRs: vPRs}
	}
}

// cachedPRsLoadedMsg is sent when cached PRs are loaded from disk.
type cachedPRsLoadedMsg struct {
	PRs     []domain.PR
	Updated time.Time
}

// loadCachedPRsCmd attempts to load PRs from the on-disk cache.
func loadCachedPRsCmd(repo domain.RepoRef) tea.Cmd {
	return func() tea.Msg {
		prs, updated, err := cache.Load(repo)
		if err != nil || len(prs) == 0 {
			return cachedPRsLoadedMsg{}
		}
		return cachedPRsLoadedMsg{PRs: prs, Updated: updated}
	}
}

// saveCacheCmd writes PRs to the on-disk cache.
func saveCacheCmd(repo domain.RepoRef, prs []domain.PR) tea.Cmd {
	return func() tea.Msg {
		_ = cache.Save(repo, prs)
		return nil
	}
}

// addInlineCommentCmd submits an inline comment on a PR.
func addInlineCommentCmd(uc *usecase.AddComment, repo domain.RepoRef, number int, input domain.InlineCommentInput) tea.Cmd {
	return func() tea.Msg {
		err := uc.Execute(context.Background(), repo, number, input)
		return views.InlineCommentAddedMsg{Err: err}
	}
}

// cloneRepoCmd clones a repo then triggers checkout.
func cloneRepoCmd(uc *usecase.SmartCheckout, repo domain.RepoRef, _ int, _ string, targetPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := uc.ExecuteClone(ctx, repo, targetPath); err != nil {
			return views.CloneDoneMsg{Path: targetPath, Err: err}
		}
		return views.CloneDoneMsg{Path: targetPath}
	}
}

// smartCheckoutCmd checks out a PR at a specific directory.
func smartCheckoutCmd(uc *usecase.SmartCheckout, repo domain.RepoRef, number int, workDir string, _ bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
		defer cancel()
		branch, err := uc.ExecuteCheckout(ctx, repo, number, workDir)
		return views.SmartCheckoutDoneMsg{Branch: branch, Path: workDir, Err: err}
	}
}

// worktreeCmd creates a worktree for a PR.
func worktreeCmd(uc *usecase.SmartCheckout, repo domain.RepoRef, number int, branch, basePath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
		defer cancel()
		wtPath, err := uc.ExecuteWorktree(ctx, repo, number, branch, basePath)
		return views.SmartCheckoutDoneMsg{Branch: branch, Path: wtPath, Err: err}
	}
}

// validateRepoCmd checks if a manually entered repo exists.
func validateRepoCmd(repo domain.RepoRef) tea.Cmd {
	return func() tea.Msg {
		err := ghcli.ValidateRepo(context.Background(), repo)
		return views.RepoValidatedMsg{Repo: repo, Err: err}
	}
}
