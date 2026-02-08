package ghcli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// ghExec runs a gh CLI command and returns stdout. It respects context cancellation.
func ghExec(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		// Map common gh errors to domain errors.
		switch {
		case cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 4:
			return nil, fmt.Errorf("%w: %s", domain.ErrNotAuthenticated, msg)
		default:
			return nil, fmt.Errorf("gh: %s", msg)
		}
	}
	return stdout.Bytes(), nil
}

// ghJSON runs a gh command and unmarshals the JSON output into dst.
func ghJSON(ctx context.Context, dst any, args ...string) error {
	out, err := ghExec(ctx, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, dst); err != nil {
		return fmt.Errorf("parsing gh output: %w", err)
	}
	return nil
}

// repoArgs returns --repo owner/name args for a command.
func repoArgs(repo domain.RepoRef) []string {
	return []string{"--repo", repo.String()}
}
