package ghcli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/indrasvat/vivecaka/internal/domain"
)

func installFakeCLIs(t *testing.T) string {
	t.Helper()

	bin := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "commands.log")
	t.Setenv("GH_TEST_LOG", logPath)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	gh := `#!/bin/sh
printf 'gh %s\n' "$*" >> "$GH_TEST_LOG"
case "$1 $2" in
  "auth status")
    exit 0
    ;;
  "api graphql")
    case "$*" in
      *pullRequests*)
        printf '{"data":{"repository":{"pullRequests":{"totalCount":7}}}}'
        ;;
      *reviewThreads*)
        cat <<'JSON'
{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"id":"thread-1","isResolved":false,"path":"a.go","line":12,"comments":{"nodes":[{"id":"c1","databaseId":101,"body":"root","createdAt":"2026-01-01T00:00:00Z","url":"https://example.test/c1","author":{"login":"alice"}}],"pageInfo":{"hasNextPage":true,"endCursor":"comment-cursor"}}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}}
JSON
        ;;
      *PullRequestReviewThread*)
        cat <<'JSON'
{"data":{"node":{"comments":{"nodes":[{"id":"c2","databaseId":102,"body":"reply","createdAt":"2026-01-01T00:01:00Z","url":"https://example.test/c2","author":{"login":"bob"}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}
JSON
        ;;
      *reviews*)
        cat <<'JSON'
{"data":{"repository":{"pullRequest":{"reviews":{"nodes":[{"id":"review-1","body":"LGTM","state":"APPROVED","submittedAt":"2026-01-01T00:02:00Z","url":"https://example.test/r1","author":{"login":"cora"}},{"id":"review-blank","body":"   ","state":"COMMENTED","submittedAt":"2026-01-01T00:03:00Z","author":{"login":"skip"}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}}
JSON
        ;;
      *comments*)
        cat <<'JSON'
{"data":{"repository":{"pullRequest":{"comments":{"nodes":[{"id":"issue-1","body":"top-level","createdAt":"2026-01-01T00:04:00Z","url":"https://example.test/i1","author":{"login":"drew"}},{"id":"issue-blank","body":"   ","createdAt":"2026-01-01T00:05:00Z","author":{"login":"skip"}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}}
JSON
        ;;
      *)
        printf '{}'
        ;;
    esac
    ;;
  "api rate_limit")
    cat <<'JSON'
{"resources":{"search":{"limit":30,"remaining":24,"reset":1767225600},"graphql":{"limit":5000,"remaining":4990,"reset":1767225600}}}
JSON
    ;;
  "api repos/owner/repo/pulls/42/comments")
    printf '{}'
    ;;
  "repo list")
    printf '[{"nameWithOwner":"owner/repo"},{"nameWithOwner":"skip-invalid"}]'
    ;;
  "repo view")
    printf '{"name":"repo"}'
    ;;
  "repo clone")
    mkdir -p "$4/.git"
    ;;
  "pr list")
    case "$*" in
      *statusCheckRollup*)
        if [ "$GH_FAIL_STATUS_ROLLUP" = "1" ]; then
          printf 'HTTP 504: 504 Gateway Timeout (https://api.github.com/graphql)\n' >&2
          exit 1
        fi
        ;;
    esac
    cat <<'JSON'
[
  {"number":1,"title":"first","author":{"login":"alice"},"state":"OPEN","headRefName":"feat/one","baseRefName":"main","labels":[{"name":"bug"}],"reviewDecision":"APPROVED","url":"https://example.test/1"},
  {"number":2,"title":"draft","author":{"login":"bob"},"state":"OPEN","isDraft":true,"headRefName":"draft","baseRefName":"main","url":"https://example.test/2"},
  {"number":3,"title":"third","author":{"login":"carol"},"state":"OPEN","headRefName":"feat/three","baseRefName":"main","reviewDecision":"CHANGES_REQUESTED","url":"https://example.test/3"}
]
JSON
    ;;
  "search prs")
    case "$*" in
      *"--review-requested @me"*)
        cat <<'JSON'
[
  {"number":11,"title":"needs review","author":{"login":"alice"},"state":"OPEN","repository":{"nameWithOwner":"team/review"},"updatedAt":"2026-07-01T10:00:00Z","createdAt":"2026-07-01T09:00:00Z","url":"https://example.test/team/review/pull/11"}
]
JSON
        ;;
      *"--assignee @me"*)
        cat <<'JSON'
[
  {"number":12,"title":"assigned fix","author":{"login":"bob"},"state":"OPEN","repository":{"nameWithOwner":"team/assigned"},"updatedAt":"2026-07-01T08:00:00Z","createdAt":"2026-07-01T07:00:00Z","url":"https://example.test/team/assigned/pull/12"}
]
JSON
        ;;
      *"--checks failure"*)
        cat <<'JSON'
[
  {"number":21,"title":"favorite failing","author":{"login":"dep"},"state":"OPEN","repository":{"nameWithOwner":"owner/fav"},"updatedAt":"2026-07-01T06:00:00Z","createdAt":"2026-07-01T05:00:00Z","url":"https://example.test/owner/fav/pull/21"}
]
JSON
        ;;
      *"--repo owner/fav"*)
        cat <<'JSON'
[
  {"number":21,"title":"favorite failing","author":{"login":"dep"},"state":"OPEN","repository":{"nameWithOwner":"owner/fav"},"updatedAt":"2026-07-01T06:00:00Z","createdAt":"2026-07-01T05:00:00Z","url":"https://example.test/owner/fav/pull/21"}
]
JSON
        ;;
      *"--owner owner"*)
        cat <<'JSON'
[
  {"number":31,"title":"owned work","author":{"login":"owner"},"state":"OPEN","repository":{"nameWithOwner":"owner/repo"},"updatedAt":"2026-07-01T04:00:00Z","createdAt":"2026-07-01T03:00:00Z","url":"https://example.test/owner/repo/pull/31"}
]
JSON
        ;;
      *)
        printf '[]'
        ;;
    esac
    ;;
  "pr view")
    cat <<'JSON'
{"number":42,"title":"full detail","author":{"login":"alice"},"state":"OPEN","headRefName":"feat/full","baseRefName":"main","headRefOid":"head-sha","baseRefOid":"base-sha","labels":[{"name":"enhancement"}],"reviewDecision":"REVIEW_REQUIRED","url":"https://example.test/42","body":"body","assignees":[{"login":"alice"}],"reviewRequests":[{"login":"bob"}],"latestReviews":[{"author":{"login":"cora"},"state":"APPROVED"}],"commits":[{"oid":"base-sha"},{"oid":"head-sha"}],"files":[{"path":"a.go","additions":2,"deletions":1,"changeType":"MODIFIED"}],"statusCheckRollup":[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS"}]}
JSON
    ;;
  "pr diff")
    cat <<'DIFF'
diff --git a/a.go b/a.go
index 0000000..1111111 100644
--- a/a.go
+++ b/a.go
@@ -1 +1,2 @@
 package a
+func added() {}
DIFF
    ;;
  "pr checks")
    printf '[{"name":"ci","status":"COMPLETED","conclusion":"SUCCESS","detailsUrl":"https://ci.test"}]'
    ;;
  "pr checkout"|"pr merge"|"pr edit"|"pr review")
    exit 0
    ;;
  *)
    printf '{}'
    ;;
esac
`
	require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte(gh), 0o755))

	git := `#!/bin/sh
printf 'git %s\n' "$*" >> "$GH_TEST_LOG"
case "$1 $2" in
  "branch --show-current")
    printf 'feat/fake\n'
    ;;
  "-C fetch"|"-C worktree"|"-C branch")
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
`
	require.NoError(t, os.WriteFile(filepath.Join(bin, "git"), []byte(git), 0o755))

	return logPath
}

func readCommandLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestAdapterCheckAndInfoUseRealPluginContract(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()

	require.NoError(t, adapter.Check())
	assert.NotEmpty(t, adapter.ghPath)
	assert.Nil(t, adapter.Init(nil))

	info := adapter.Info()
	assert.Equal(t, "ghcli", info.Name)
	assert.Contains(t, info.Provides, "pr-reader")
	assert.Contains(t, readCommandLog(t, logPath), "gh auth status")
}

func TestAdapterListPRsBuildsFiltersAndPaginatesAfterDraftFiltering(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	prs, err := adapter.ListPRs(context.Background(), repo, domain.ListOpts{
		State:   domain.PRStateOpen,
		Author:  "alice",
		Labels:  []string{"bug", "backend"},
		Search:  "status:success",
		Draft:   domain.DraftExclude,
		Page:    2,
		PerPage: 1,
	})

	require.NoError(t, err)
	require.Len(t, prs, 1)
	assert.Equal(t, 3, prs[0].Number, "drafts are filtered before page slicing")
	assert.Equal(t, domain.ReviewChangesRequested, prs[0].Review.State)

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "gh pr list --json")
	assert.Contains(t, log, "--repo owner/repo")
	assert.Contains(t, log, "--state open")
	assert.Contains(t, log, "--author alice")
	assert.Contains(t, log, "--label bug")
	assert.Contains(t, log, "--label backend")
	assert.Contains(t, log, "--search status:success")
	assert.Contains(t, log, "--limit 2")
	assert.NotContains(t, log, "statusCheckRollup", "paginated list should use the light field set")
}

func TestAdapterListPRsPreservesStatusRollupOnInitialLoad(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	prs, err := adapter.ListPRs(context.Background(), repo, domain.ListOpts{
		State:   domain.PRStateOpen,
		Page:    1,
		PerPage: 50,
	})

	require.NoError(t, err)
	require.Len(t, prs, 3)

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "gh pr list --json")
	assert.Contains(t, log, "--limit 50")
	assert.Contains(t, log, "statusCheckRollup", "initial list should preserve CI rollups when GitHub serves them")
}

func TestAdapterListPRsFallsBackWhenStatusRollupTimesOut(t *testing.T) {
	logPath := installFakeCLIs(t)
	t.Setenv("GH_FAIL_STATUS_ROLLUP", "1")
	adapter := New()
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	prs, err := adapter.ListPRs(context.Background(), repo, domain.ListOpts{
		State:   domain.PRStateOpen,
		Page:    1,
		PerPage: 10,
	})

	require.NoError(t, err)
	require.Len(t, prs, 3)

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "statusCheckRollup", "small list should first try CI rollups")
	assert.Contains(t, log, "--limit 10")
	assert.Equal(t, 2, strings.Count(log, "gh pr list --json"), "timeout should be retried with light fields")
}

func TestAdapterGetInboxUsesSearchSourcesAndMergesDuplicates(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()

	result, err := adapter.GetInbox(context.Background(), domain.InboxQuery{
		Username:          "owner",
		HomeRepo:          domain.RepoRef{Owner: "owner", Name: "repo"},
		Favorites:         []domain.RepoRef{{Owner: "owner", Name: "fav"}},
		OwnedOwner:        "owner",
		IncludeOwnedRepos: true,
		Limit:             25,
		SourceTimeout:     time.Second,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 24, result.Rate.SearchRemaining)
	require.Len(t, result.Items, 4)
	require.Len(t, result.Sources, 7)
	assert.Equal(t, []string{"review", "assigned", "home", "favs", "favs ci", "owned", "owned ci"}, []string{
		result.Sources[0].Label,
		result.Sources[1].Label,
		result.Sources[2].Label,
		result.Sources[3].Label,
		result.Sources[4].Label,
		result.Sources[5].Label,
		result.Sources[6].Label,
	})

	var favorite *domain.InboxItem
	var assigned *domain.InboxItem
	for i := range result.Items {
		if result.Items[i].Repo == (domain.RepoRef{Owner: "owner", Name: "fav"}) {
			favorite = &result.Items[i]
		}
		if result.Items[i].Repo == (domain.RepoRef{Owner: "team", Name: "assigned"}) {
			assigned = &result.Items[i]
		}
	}
	require.NotNil(t, favorite)
	assert.Equal(t, domain.CIFail, favorite.CI)
	assert.ElementsMatch(t, []domain.InboxSource{domain.InboxSourceFavorite, domain.InboxSourceOwned}, favorite.Sources)
	require.NotNil(t, assigned)
	assert.Equal(t, []domain.InboxSource{domain.InboxSourceAssigned}, assigned.Sources)
	assert.Equal(t, domain.ReviewNone, assigned.Review.State)

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "gh api rate_limit")
	assert.Contains(t, log, "gh search prs --review-requested @me")
	assert.Contains(t, log, "gh search prs --assignee @me")
	assert.Contains(t, log, "gh search prs --owner owner")
	assert.Contains(t, log, "gh search prs --state open --archived=false --limit 25 --repo owner/fav")
	assert.Contains(t, log, "gh search prs --state open --archived=false --limit 25 --repo owner/repo")
}

func TestBuildInboxSearchJobsHonorsRateLowWatermark(t *testing.T) {
	jobs := buildInboxSearchJobs(domain.InboxQuery{
		HomeRepo:          domain.RepoRef{Owner: "owner", Name: "repo"},
		Favorites:         []domain.RepoRef{{Owner: "owner", Name: "fav"}},
		OwnedOwner:        "owner",
		IncludeOwnedRepos: true,
		Limit:             25,
		RateLowWatermark:  25,
	}, domain.InboxRateLimit{SearchRemaining: 24})

	var labels []string
	for _, job := range jobs {
		labels = append(labels, job.label)
	}
	assert.Equal(t, []string{"review", "assigned", "home"}, labels)
}

func TestAdapterInboxInsightFetchesSelectedRowDeltas(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	insight, err := adapter.GetInboxInsight(context.Background(), domain.InboxInsightQuery{
		Repo:              repo,
		Number:            42,
		LastReviewHeadSHA: "base-sha",
		LastReviewFiles:   map[string]string{"old.go": "digest"},
	})

	require.NoError(t, err)
	assert.Equal(t, repo, insight.Repo)
	assert.Equal(t, 42, insight.Number)
	assert.Equal(t, "head-sha", insight.HeadSHA)
	assert.Equal(t, 1, insight.CommitDelta)
	assert.Equal(t, 1, insight.FileDelta)
	assert.Equal(t, 1, insight.UnresolvedThreads)
	assert.True(t, insight.HasReviewBaseline)

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "gh pr view 42 --json headRefOid,files,commits --repo owner/repo")
	assert.Contains(t, log, "reviewThreads(first: 100")
}

func TestAdapterReadCommandsMapGHOutput(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	count, err := adapter.GetPRCount(context.Background(), repo, domain.PRStateMerged)
	require.NoError(t, err)
	assert.Equal(t, 7, count)

	detail, err := adapter.GetPR(context.Background(), repo, 42)
	require.NoError(t, err)
	assert.Equal(t, "full detail", detail.Title)
	assert.Equal(t, "head-sha", detail.Branch.HeadSHA)
	assert.Equal(t, []string{"alice"}, detail.Assignees)
	require.Len(t, detail.Reviewers, 2)
	assert.Equal(t, domain.ReviewPending, detail.Reviewers[0].State)
	assert.Equal(t, domain.ReviewApproved, detail.Reviewers[1].State)
	require.Len(t, detail.Files, 1)
	assert.Equal(t, "modified", detail.Files[0].Status)

	diff, err := adapter.GetDiff(context.Background(), repo, 42)
	require.NoError(t, err)
	require.Len(t, diff.Files, 1)
	assert.Equal(t, "a.go", diff.Files[0].Path)

	checks, err := adapter.GetChecks(context.Background(), repo, 42)
	require.NoError(t, err)
	require.Len(t, checks, 1)
	assert.Equal(t, domain.CIPass, checks[0].Status)

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "pullRequests(states: MERGED)")
	assert.Contains(t, log, "gh pr view 42")
	assert.Contains(t, log, "gh pr diff 42 --repo owner/repo")
	assert.Contains(t, log, "gh pr checks 42")
}

func TestAdapterGraphQLDiscussionCommandsPaginateAndMapThreads(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	threads, err := adapter.GetComments(context.Background(), repo, 42)
	require.NoError(t, err)
	require.Len(t, threads, 1)
	assert.Equal(t, "thread-1", threads[0].ThreadID)
	assert.Equal(t, "101", threads[0].ReplyToID)
	assert.Equal(t, "a.go", threads[0].Path)
	assert.Equal(t, 12, threads[0].Line)
	require.Len(t, threads[0].Comments, 2)
	assert.Equal(t, "alice", threads[0].Comments[0].Author)
	assert.Equal(t, "bob", threads[0].Comments[1].Author)

	discussion, err := adapter.GetDiscussion(context.Background(), repo, 42)
	require.NoError(t, err)
	require.Len(t, discussion, 2)
	assert.Equal(t, domain.DiscussionComment, discussion[0].Kind)
	assert.Equal(t, "top-level", discussion[0].Comments[0].Body)
	assert.Equal(t, domain.DiscussionReview, discussion[1].Kind)
	assert.Equal(t, domain.ReviewApproved, discussion[1].ReviewState)

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "reviewThreads(first: 100")
	assert.Contains(t, log, `node(id: "thread-1")`)
	assert.Contains(t, log, "comments(first: 100")
	assert.Contains(t, log, "reviews(first: 100")
}

func TestAdapterWriteAndReviewCommandsPreserveUserIntent(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	branch, err := adapter.Checkout(context.Background(), repo, 42)
	require.NoError(t, err)
	assert.Equal(t, "feat/fake", branch)

	require.NoError(t, adapter.Merge(context.Background(), repo, 42, domain.MergeOpts{
		Method:        "squash",
		DeleteBranch:  true,
		CommitMessage: "ship it",
	}))
	require.NoError(t, adapter.Merge(context.Background(), repo, 43, domain.MergeOpts{Method: "rebase"}))
	require.NoError(t, adapter.Merge(context.Background(), repo, 44, domain.MergeOpts{}))
	require.NoError(t, adapter.UpdateLabels(context.Background(), repo, 42, []string{"bug", "urgent"}))

	require.NoError(t, adapter.SubmitReview(context.Background(), repo, 42, domain.Review{
		Action: domain.ReviewActionApprove,
		Body:   "looks good",
	}))
	require.NoError(t, adapter.SubmitReview(context.Background(), repo, 42, domain.Review{Action: domain.ReviewActionRequestChanges}))
	require.NoError(t, adapter.SubmitReview(context.Background(), repo, 42, domain.Review{Action: domain.ReviewActionComment}))
	require.Error(t, adapter.SubmitReview(context.Background(), repo, 42, domain.Review{Action: "dismiss"}))

	require.NoError(t, adapter.AddComment(context.Background(), repo, 42, domain.InlineCommentInput{
		Body: "please tighten this", Path: "a.go", Line: 9, Side: "RIGHT", CommitID: "abc123", InReplyTo: "1001",
	}))
	require.NoError(t, adapter.ResolveThread(context.Background(), repo, "thread-1"))

	log := readCommandLog(t, logPath)
	for _, want := range []string{
		"gh pr checkout 42 --repo owner/repo",
		"git branch --show-current",
		"gh pr merge 42 --repo owner/repo --squash --delete-branch --body ship it",
		"gh pr merge 43 --repo owner/repo --rebase",
		"gh pr merge 44 --repo owner/repo --merge",
		"gh pr edit 42 --repo owner/repo --add-label bug --add-label urgent",
		"gh pr review 42 --repo owner/repo --approve --body looks good",
		"gh pr review 42 --repo owner/repo --request-changes",
		"gh pr review 42 --repo owner/repo --comment",
		"gh api repos/owner/repo/pulls/42/comments --method POST",
		"--raw-field in_reply_to=1001",
		"gh api graphql -f query=mutation",
	} {
		assert.Contains(t, strings.ReplaceAll(log, "\n", " "), want)
	}
}

func TestAdapterRepoManagementCommandsHandleLocalState(t *testing.T) {
	logPath := installFakeCLIs(t)
	adapter := New()
	repo := domain.RepoRef{Owner: "owner", Name: "repo"}

	branch, err := adapter.CheckoutAt(context.Background(), repo, 42, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, "feat/fake", branch)

	existing := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(existing, ".git"), 0o755))
	require.NoError(t, adapter.CloneRepo(context.Background(), repo, existing))

	corrupt := filepath.Join(t.TempDir(), "repo")
	require.NoError(t, os.MkdirAll(corrupt, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(corrupt, "stale.txt"), []byte("old"), 0o644))
	require.NoError(t, adapter.CloneRepo(context.Background(), repo, corrupt))
	assert.DirExists(t, filepath.Join(corrupt, ".git"))
	assert.NoFileExists(t, filepath.Join(corrupt, "stale.txt"))

	require.NoError(t, adapter.CreateWorktree(context.Background(), t.TempDir(), 42, "feat/fake", filepath.Join(t.TempDir(), "wt")))

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "gh pr checkout 42 --repo owner/repo")
	assert.Contains(t, log, "git -C "+existing+" fetch --all")
	assert.Contains(t, log, "gh repo clone owner/repo "+corrupt)
	assert.Contains(t, log, "fetch origin pull/42/head:pr-42")
	assert.Contains(t, log, "worktree add")
}

func TestStandaloneRepoAndUserDiscoveryCommands(t *testing.T) {
	logPath := installFakeCLIs(t)

	repos, err := ListUserRepos(context.Background(), 0)
	require.NoError(t, err)
	assert.Equal(t, []domain.RepoRef{{Owner: "owner", Name: "repo"}}, repos)

	require.NoError(t, ValidateRepo(context.Background(), domain.RepoRef{Owner: "owner", Name: "repo"}))

	log := readCommandLog(t, logPath)
	assert.Contains(t, log, "gh repo list --json nameWithOwner -L 20")
	assert.Contains(t, log, "gh repo view owner/repo --json name")
}
