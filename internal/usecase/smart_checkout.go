package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/repolocator"
)

// CheckoutStrategy describes the approach for checking out a PR.
type CheckoutStrategy int

const (
	// StrategyLocal means CWD is the correct repo — checkout in place.
	StrategyLocal CheckoutStrategy = iota
	// StrategyKnownPath means known-repos has a valid path for this repo.
	StrategyKnownPath
	// StrategyNeedsClone means no local clone exists.
	StrategyNeedsClone
)

// CheckoutContext describes the current checkout situation.
type CheckoutContext struct {
	BrowsingRepo domain.RepoRef
	CWDRepo      domain.RepoRef // zero value if CWD is not a git repo
	CWDPath      string         // os.Getwd() result
}

// CheckoutPlan describes what the app should do for checkout.
type CheckoutPlan struct {
	Strategy       CheckoutStrategy
	TargetPath     string // where checkout will happen
	KnownRepo      bool   // true if path came from known-repos
	CacheClonePath string // managed cache path (for dialog display)
}

// SmartCheckout orchestrates the checkout decision cascade.
// Depends on domain.RepoManager (NOT PRWriter) and repolocator.Locator.
type SmartCheckout struct {
	repoMgr domain.RepoManager
	locator *repolocator.Locator
}

// NewSmartCheckout creates a new SmartCheckout use case.
func NewSmartCheckout(repoMgr domain.RepoManager, locator *repolocator.Locator) *SmartCheckout {
	return &SmartCheckout{
		repoMgr: repoMgr,
		locator: locator,
	}
}

// Plan determines the checkout strategy based on pre-validated context.
// knownPath and knownPathValid should come from locator.Validate() called
// asynchronously by the TUI before invoking Plan().
func (uc *SmartCheckout) Plan(ctx CheckoutContext, knownPath string, knownPathValid bool) CheckoutPlan {
	plan := CheckoutPlan{
		CacheClonePath: uc.locator.CacheClonePath(ctx.BrowsingRepo),
	}

	// Case 1: CWD is the correct repo.
	if reposMatch(ctx.BrowsingRepo, ctx.CWDRepo) {
		plan.Strategy = StrategyLocal
		plan.TargetPath = ctx.CWDPath
		return plan
	}

	// Case 2: Known-repos has a valid path.
	if knownPathValid && knownPath != "" {
		plan.Strategy = StrategyKnownPath
		plan.TargetPath = knownPath
		plan.KnownRepo = true
		return plan
	}

	// Case 3: No local clone found.
	plan.Strategy = StrategyNeedsClone
	return plan
}

// ExecuteClone clones a repo and registers it in known-repos.
func (uc *SmartCheckout) ExecuteClone(ctx context.Context, repo domain.RepoRef, targetPath string) error {
	if err := uc.repoMgr.CloneRepo(ctx, repo, targetPath); err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	// Register the new clone so future checkouts skip the dialog.
	_ = uc.locator.Register(repo, targetPath, "cloned")
	return nil
}

// ExecuteCheckout checks out a PR in the specified directory.
func (uc *SmartCheckout) ExecuteCheckout(ctx context.Context, repo domain.RepoRef, number int, workDir string) (string, error) {
	return uc.repoMgr.CheckoutAt(ctx, repo, number, workDir)
}

// ExecuteWorktree creates a worktree for a PR branch.
// Returns the full worktree path.
func (uc *SmartCheckout) ExecuteWorktree(ctx context.Context, repo domain.RepoRef, number int, branch, basePath string) (string, error) {
	// Generate worktree path: .worktrees/pr-N-branch-name
	safeBranch := sanitizeBranchName(branch)
	wtDir := fmt.Sprintf("pr-%d-%s", number, safeBranch)
	wtPath := filepath.Join(basePath, ".worktrees", wtDir)

	if err := uc.repoMgr.CreateWorktree(ctx, basePath, number, branch, wtPath); err != nil {
		return "", fmt.Errorf("worktree: %w", err)
	}
	return wtPath, nil
}

// WorktreePath returns the path that a worktree would be created at.
func (uc *SmartCheckout) WorktreePath(number int, branch, basePath string) string {
	safeBranch := sanitizeBranchName(branch)
	wtDir := fmt.Sprintf("pr-%d-%s", number, safeBranch)
	return filepath.Join(basePath, ".worktrees", wtDir)
}

func reposMatch(a, b domain.RepoRef) bool {
	return strings.EqualFold(a.Owner, b.Owner) && strings.EqualFold(a.Name, b.Name)
}

func sanitizeBranchName(branch string) string {
	// Replace slashes and other problematic characters with dashes.
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return r.Replace(branch)
}
