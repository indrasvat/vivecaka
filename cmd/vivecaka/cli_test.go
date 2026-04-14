package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestParseRepoRef(t *testing.T) {
	t.Parallel()

	repo, err := parseRepoRef("indrasvat/vivecaka")
	require.NoError(t, err)
	assert.Equal(t, "indrasvat", repo.Owner)
	assert.Equal(t, "vivecaka", repo.Name)
}

func TestParseRepoRefRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	_, err := parseRepoRef("not-a-repo")
	require.Error(t, err)
}

func TestOptionsFromEnv(t *testing.T) {
	t.Parallel()

	opts, err := optionsFromEnv(func(key string) string {
		switch key {
		case debugEnvVar:
			return "true"
		case repoEnvVar:
			return "indrasvat/vivecaka"
		default:
			return ""
		}
	})
	require.NoError(t, err)
	assert.True(t, opts.debug)
	assert.Equal(t, "indrasvat", opts.repo.Owner)
	assert.Equal(t, repoEnvVar, opts.repoSource)
}

func TestOptionsFromEnvRejectsInvalidRepo(t *testing.T) {
	t.Parallel()

	_, err := optionsFromEnv(func(key string) string {
		if key == repoEnvVar {
			return "bad"
		}
		return ""
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), repoEnvVar)
}

func TestRootCommandHelpDoesNotRunApp(t *testing.T) {
	t.Parallel()

	cmd, stdout, _, called, _ := newTestRootCommand(cliOptions{})
	cmd.SetArgs([]string{"--help"})

	require.NoError(t, cmd.Execute())
	assert.False(t, *called)
	assert.Contains(t, stdout.String(), "Usage")
	assert.Contains(t, stdout.String(), "--repo owner/name")
	assert.Contains(t, stdout.String(), repoEnvVar)
}

func TestRootCommandVersionIgnoresFlagOrder(t *testing.T) {
	t.Parallel()

	cmd, stdout, _, called, _ := newTestRootCommand(cliOptions{})
	cmd.SetArgs([]string{"--repo", "indrasvat/vivecaka", "--version"})

	require.NoError(t, cmd.Execute())
	assert.False(t, *called)
	assert.Contains(t, stdout.String(), "vivecaka ")
	assert.Contains(t, stdout.String(), "commit:")
}

func TestRootCommandPassesParsedRepoToRun(t *testing.T) {
	t.Parallel()

	cmd, _, _, called, received := newTestRootCommand(cliOptions{})
	cmd.SetArgs([]string{"--repo", "indrasvat/vivecaka"})

	require.NoError(t, cmd.Execute())
	assert.True(t, *called)
	assert.Equal(t, "indrasvat", received.repo.Owner)
	assert.Equal(t, "vivecaka", received.repo.Name)
	assert.Equal(t, "flag", received.repoSource)
}

func TestRootCommandFlagOverridesEnvRepo(t *testing.T) {
	t.Parallel()

	cmd, _, _, called, received := newTestRootCommand(cliOptions{
		repo:       mustParseRepoRef(t, "owner/from-env"),
		repoSource: repoEnvVar,
	})
	cmd.SetArgs([]string{"--repo", "indrasvat/vivecaka"})

	require.NoError(t, cmd.Execute())
	assert.True(t, *called)
	assert.Equal(t, "indrasvat/vivecaka", received.repo.String())
	assert.Equal(t, "flag", received.repoSource)
}

func TestRootCommandRejectsUnexpectedArgs(t *testing.T) {
	t.Parallel()

	cmd, _, stderr, called, _ := newTestRootCommand(cliOptions{})
	cmd.SetArgs([]string{"unexpected"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.False(t, *called)
	assert.Contains(t, err.Error(), "unknown command")
	assert.Empty(t, stderr.String())
}

func newTestRootCommand(opts cliOptions) (*cobra.Command, *bytes.Buffer, *bytes.Buffer, *bool, *cliOptions) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	called := false
	var received cliOptions

	cmd := newRootCommand(opts, &stdout, &stderr, func(parsed cliOptions) error {
		called = true
		received = parsed
		return nil
	})
	return cmd, &stdout, &stderr, &called, &received
}

func mustParseRepoRef(t *testing.T, value string) domain.RepoRef {
	t.Helper()
	repo, err := parseRepoRef(value)
	require.NoError(t, err)
	return repo
}
