package ghcli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// Compile-time check that Adapter implements domain.RepoManager.
var _ domain.RepoManager = (*Adapter)(nil)

// CheckoutAt checks out a PR branch in the specified working directory.
// If workDir is "", uses the process CWD (identical to PRWriter.Checkout behavior).
func (a *Adapter) CheckoutAt(ctx context.Context, repo domain.RepoRef, number int, workDir string) (string, error) {
	args := []string{"pr", "checkout", fmt.Sprintf("%d", number)}
	args = append(args, repoArgs(repo)...)

	cmd := exec.CommandContext(ctx, "gh", args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("checking out PR #%d: %s: %w", number, strings.TrimSpace(string(out)), err)
	}

	// gh pr checkout outputs branch info to stderr, not stdout.
	// Get the actual branch name via git after checkout succeeds.
	gitCmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	if workDir != "" {
		gitCmd.Dir = workDir
	}
	out, err := gitCmd.Output()
	if err != nil {
		// Fallback to PR number if git branch detection fails — checkout still succeeded.
		return fmt.Sprintf("PR #%d", number), nil //nolint:nilerr // fallback to PR number if branch detection fails
	}
	return strings.TrimSpace(string(out)), nil
}

// CloneRepo clones a repository to the specified local path.
// If the target path exists and is a valid clone, it skips cloning and fetches instead.
func (a *Adapter) CloneRepo(ctx context.Context, repo domain.RepoRef, targetPath string) error {
	// Check if target already exists and is a valid git repo.
	if info, err := os.Stat(filepath.Join(targetPath, ".git")); err == nil && info.IsDir() {
		// Already cloned — just fetch latest.
		cmd := exec.CommandContext(ctx, "git", "-C", targetPath, "fetch", "--all")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("fetching in existing clone: %s: %w", strings.TrimSpace(string(out)), err)
		}
		return nil
	}

	// If target exists but is corrupted (no .git), remove it.
	if _, err := os.Stat(targetPath); err == nil {
		if err := os.RemoveAll(targetPath); err != nil {
			return fmt.Errorf("removing corrupted clone: %w", err)
		}
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}

	cmd := exec.CommandContext(ctx, "gh", "repo", "clone", repo.String(), targetPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Clean up partial clone on failure.
		_ = os.RemoveAll(targetPath)
		return fmt.Errorf("cloning %s: %s: %w", repo, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// CreateWorktree creates a git worktree for a PR branch at the given path.
// It fetches the PR ref first (needed for fork branches), then creates the worktree.
// Uses a unique local branch name (pr-<number>) to avoid collisions with existing
// branches — e.g. a fork PR named "main" would otherwise overwrite the local main.
func (a *Adapter) CreateWorktree(ctx context.Context, repoPath string, number int, branch, worktreePath string) error {
	// Use a unique local branch name to avoid colliding with existing branches.
	localBranch := fmt.Sprintf("pr-%d", number)

	// Step 1: Fetch the PR ref into our unique local branch.
	fetchCmd := exec.CommandContext(ctx, "git", "-C", repoPath,
		"fetch", "origin", fmt.Sprintf("pull/%d/head:%s", number, localBranch))
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fetching PR #%d ref: %s: %w", number, strings.TrimSpace(string(out)), err)
	}

	// Step 2: Create the worktree.
	wtCmd := exec.CommandContext(ctx, "git", "-C", repoPath,
		"worktree", "add", worktreePath, localBranch)
	if out, err := wtCmd.CombinedOutput(); err != nil {
		// Cleanup on failure: remove worktree and the branch we created.
		cleanCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "worktree", "remove", worktreePath)
		_ = cleanCmd.Run()
		delCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "-D", localBranch)
		_ = delCmd.Run()
		return fmt.Errorf("creating worktree: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
