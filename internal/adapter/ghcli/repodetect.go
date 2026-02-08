package ghcli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// DetectRepo attempts to detect the current repo from the git remote URL.
func DetectRepo(ctx context.Context) (domain.RepoRef, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return domain.RepoRef{}, domain.ErrNotFound
	}

	url := strings.TrimSpace(string(out))
	ref, ok := ParseRemoteURL(url)
	if !ok {
		return domain.RepoRef{}, domain.ErrNotFound
	}
	return ref, nil
}

var (
	// SSH: git@github.com:owner/repo.git
	sshPattern = regexp.MustCompile(`^git@github\.com:([^/]+)/(.+?)(?:\.git)?$`)
	// HTTPS: https://github.com/owner/repo.git
	httpsPattern = regexp.MustCompile(`^https?://github\.com/([^/]+)/(.+?)(?:\.git)?$`)
)

// ListUserRepos fetches the authenticated user's repos via `gh repo list`.
func ListUserRepos(ctx context.Context, limit int) ([]domain.RepoRef, error) {
	if limit <= 0 {
		limit = 20
	}
	cmd := exec.CommandContext(ctx, "gh", "repo", "list",
		"--json", "nameWithOwner",
		"-L", fmt.Sprintf("%d", limit),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var items []struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}

	repos := make([]domain.RepoRef, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item.NameWithOwner, "/", 2)
		if len(parts) == 2 {
			repos = append(repos, domain.RepoRef{Owner: parts[0], Name: parts[1]})
		}
	}
	return repos, nil
}

// ValidateRepo checks if a repo exists via `gh repo view`.
func ValidateRepo(ctx context.Context, repo domain.RepoRef) error {
	cmd := exec.CommandContext(ctx, "gh", "repo", "view",
		repo.String(), "--json", "name",
	)
	_, err := cmd.Output()
	return err
}

// ParseRemoteURL extracts owner/name from a GitHub remote URL.
// Supports both SSH and HTTPS formats.
func ParseRemoteURL(url string) (domain.RepoRef, bool) {
	url = strings.TrimSpace(url)

	if m := sshPattern.FindStringSubmatch(url); m != nil {
		return domain.RepoRef{Owner: m[1], Name: m[2]}, true
	}
	if m := httpsPattern.FindStringSubmatch(url); m != nil {
		return domain.RepoRef{Owner: m[1], Name: m[2]}, true
	}
	return domain.RepoRef{}, false
}
