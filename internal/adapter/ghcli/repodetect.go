package ghcli

import (
	"context"
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
