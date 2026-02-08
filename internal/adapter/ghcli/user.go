package ghcli

import (
	"context"
	"os/exec"
	"strings"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// DetectUser fetches the current GitHub username via gh CLI.
func DetectUser(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "api", "user", "--jq", ".login")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	login := strings.TrimSpace(string(out))
	if login == "" {
		return "", domain.ErrNotFound
	}
	return login, nil
}
