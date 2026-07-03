package ghcli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
	"github.com/indrasvat/vivecaka/internal/reviewprogress"
)

const searchPRFields = "repository,number,title,author,state,isDraft,updatedAt,createdAt,url"
const inboxInsightFields = "headRefOid,files,commits"

type ghSearchPR struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	Author     ghActor   `json:"author"`
	State      string    `json:"state"`
	IsDraft    bool      `json:"isDraft"`
	UpdatedAt  time.Time `json:"updatedAt"`
	CreatedAt  time.Time `json:"createdAt"`
	URL        string    `json:"url"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
}

type ghRateLimit struct {
	Resources struct {
		Search  ghRateResource `json:"search"`
		GraphQL ghRateResource `json:"graphql"`
	} `json:"resources"`
}

type ghRateResource struct {
	Limit     int `json:"limit"`
	Remaining int `json:"remaining"`
	Reset     int `json:"reset"`
}

type ghInboxCommit struct {
	OID string `json:"oid"`
}

type ghInboxInsightPR struct {
	HeadRefOID string          `json:"headRefOid"`
	Files      []ghFile        `json:"files"`
	Commits    []ghInboxCommit `json:"commits"`
}

type inboxSearchJob struct {
	label  string
	source domain.InboxSource
	args   []string
	ci     domain.CIStatus
}

// GetInbox fetches the attention inbox with bounded, source-level fanout.
func (a *Adapter) GetInbox(ctx context.Context, query domain.InboxQuery) (*domain.InboxResult, error) {
	rate := a.getInboxRateLimit(ctx)
	jobs := buildInboxSearchJobs(query, rate)
	if len(jobs) == 0 {
		return &domain.InboxResult{Rate: rate}, nil
	}

	var (
		mu       sync.Mutex
		byKey    = make(map[string]domain.InboxItem)
		statuses []domain.InboxSourceStatus
		wg       sync.WaitGroup
	)
	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			items, err := a.runInboxSearchJob(ctx, query.SourceTimeout, job)
			status := domain.InboxSourceStatus{
				Source:  job.source,
				Label:   job.label,
				Count:   len(items),
				Elapsed: time.Since(start),
			}
			if err != nil {
				status.Err = err.Error()
			}

			mu.Lock()
			defer mu.Unlock()
			statuses = append(statuses, status)
			if err != nil {
				return
			}
			for _, item := range items {
				key := inboxItemKey(item)
				existing, ok := byKey[key]
				if !ok {
					byKey[key] = item
					continue
				}
				existing.Sources = mergeInboxSources(existing.Sources, item.Sources)
				if existing.CI == domain.CINone && item.CI != domain.CINone {
					existing.CI = item.CI
				}
				if existing.Review.State == domain.ReviewNone && item.Review.State != domain.ReviewNone {
					existing.Review = item.Review
				}
				byKey[key] = existing
			}
		}()
	}
	wg.Wait()

	items := make([]domain.InboxItem, 0, len(byKey))
	for _, item := range byKey {
		items = append(items, item)
	}
	sort.SliceStable(statuses, func(i, j int) bool {
		return inboxStatusOrder(statuses[i].Label) < inboxStatusOrder(statuses[j].Label)
	})
	return &domain.InboxResult{Items: items, Sources: statuses, Rate: rate}, nil
}

// GetInboxInsight fetches lazy selected-row delta data for a single PR.
func (a *Adapter) GetInboxInsight(ctx context.Context, query domain.InboxInsightQuery) (*domain.InboxInsight, error) {
	args := []string{"pr", "view", fmt.Sprintf("%d", query.Number), "--json", inboxInsightFields}
	args = append(args, repoArgs(query.Repo)...)

	var pr ghInboxInsightPR
	if err := ghJSON(ctx, &pr, args...); err != nil {
		return nil, fmt.Errorf("getting inbox insight for PR #%d: %w", query.Number, err)
	}

	insight := &domain.InboxInsight{
		Repo:              query.Repo,
		Number:            query.Number,
		HeadSHA:           pr.HeadRefOID,
		CommitDelta:       countCommitsSince(pr.Commits, query.LastReviewHeadSHA),
		FileDelta:         countFilesSince(pr.Files, query.LastReviewFiles),
		HasReviewBaseline: query.LastReviewHeadSHA != "" || len(query.LastReviewFiles) > 0,
		LocalState:        query.LocalState,
		FetchedAt:         time.Now(),
	}

	unresolved, err := a.countUnresolvedReviewThreads(ctx, query.Repo, query.Number)
	if err == nil {
		insight.UnresolvedThreads = unresolved
	}
	return insight, nil
}

func countCommitsSince(commits []ghInboxCommit, baseline string) int {
	if baseline == "" {
		return 0
	}
	for i, commit := range commits {
		if commit.OID == baseline {
			return len(commits) - i - 1
		}
	}
	return len(commits)
}

func countFilesSince(files []ghFile, baseline map[string]string) int {
	if len(baseline) == 0 {
		return len(files)
	}
	var count int
	for _, file := range files {
		current := reviewprogress.FallbackDigest(domain.FileChange{
			Path:      file.Path,
			Additions: file.Additions,
			Deletions: file.Deletions,
			Status:    mapFileStatus(file.ChangeType),
		})
		if baseline[file.Path] != current {
			count++
		}
	}
	return count
}

func (a *Adapter) countUnresolvedReviewThreads(ctx context.Context, repo domain.RepoRef, number int) (int, error) {
	var count int
	cursor := ""
	for {
		page, err := fetchReviewThreadsPage(ctx, repo, number, cursor)
		if err != nil {
			return count, err
		}
		for _, thread := range page.Nodes {
			if !thread.IsResolved {
				count++
			}
		}
		if !page.PageInfo.HasNextPage {
			return count, nil
		}
		cursor = page.PageInfo.EndCursor
	}
}

func inboxStatusOrder(label string) int {
	switch label {
	case "review":
		return 0
	case "assigned":
		return 1
	case "home":
		return 2
	case "favs":
		return 3
	case "favs ci":
		return 4
	case "owned":
		return 5
	case "owned ci":
		return 6
	default:
		return 99
	}
}

func (a *Adapter) runInboxSearchJob(ctx context.Context, timeout time.Duration, job inboxSearchJob) ([]domain.InboxItem, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var prs []ghSearchPR
	args := append([]string{"search", "prs"}, job.args...)
	args = append(args, "--json", searchPRFields)
	if err := ghJSON(ctx, &prs, args...); err != nil {
		return nil, err
	}

	items := make([]domain.InboxItem, 0, len(prs))
	for _, pr := range prs {
		item, ok := toInboxItem(pr, job.source, job.ci)
		if ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func buildInboxSearchJobs(query domain.InboxQuery, rate domain.InboxRateLimit) []inboxSearchJob {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	jobs := []inboxSearchJob{
		{
			label:  "review",
			source: domain.InboxSourceAttention,
			args:   []string{"--review-requested", "@me", "--state", "open", "--archived=false", "--limit", fmt.Sprintf("%d", min(limit, 30))},
		},
		{
			label:  "assigned",
			source: domain.InboxSourceAssigned,
			args:   []string{"--assignee", "@me", "--state", "open", "--archived=false", "--limit", fmt.Sprintf("%d", min(limit, 30))},
		},
	}

	lowWatermark := query.RateLowWatermark
	if lowWatermark <= 0 {
		lowWatermark = 6
	}
	lowSearchBudget := rate.SearchRemaining > 0 && rate.SearchRemaining < lowWatermark

	if query.HomeRepo.Owner != "" && query.HomeRepo.Name != "" {
		jobs = append(jobs, inboxSearchJob{
			label:  "home",
			source: domain.InboxSourceHome,
			args:   append([]string{"--state", "open", "--archived=false", "--limit", fmt.Sprintf("%d", min(limit, 30))}, repoArgs(query.HomeRepo)...),
		})
	}

	if lowSearchBudget {
		return jobs
	}

	if query.IncludeOwnedRepos && query.OwnedOwner != "" {
		jobs = append(jobs,
			inboxSearchJob{
				label:  "owned",
				source: domain.InboxSourceOwned,
				args:   []string{"--owner", query.OwnedOwner, "--state", "open", "--archived=false", "--limit", fmt.Sprintf("%d", limit)},
			},
			inboxSearchJob{
				label:  "owned ci",
				source: domain.InboxSourceOwned,
				args:   []string{"--owner", query.OwnedOwner, "--state", "open", "--archived=false", "--checks", "failure", "--limit", fmt.Sprintf("%d", min(limit, 30))},
				ci:     domain.CIFail,
			},
		)
	}

	if len(query.Favorites) > 0 {
		args := []string{"--state", "open", "--archived=false", "--limit", fmt.Sprintf("%d", limit)}
		for _, repo := range query.Favorites {
			if repo.Owner == "" || repo.Name == "" {
				continue
			}
			args = append(args, repoArgs(repo)...)
		}
		if len(args) > 6 {
			jobs = append(jobs, inboxSearchJob{
				label:  "favs",
				source: domain.InboxSourceFavorite,
				args:   args,
			})
		}

		failArgs := []string{"--state", "open", "--archived=false", "--checks", "failure", "--limit", fmt.Sprintf("%d", min(limit, 30))}
		for _, repo := range query.Favorites {
			if repo.Owner == "" || repo.Name == "" {
				continue
			}
			failArgs = append(failArgs, repoArgs(repo)...)
		}
		if len(failArgs) > 8 {
			jobs = append(jobs, inboxSearchJob{
				label:  "favs ci",
				source: domain.InboxSourceFavorite,
				args:   failArgs,
				ci:     domain.CIFail,
			})
		}
	}

	return jobs
}

func (a *Adapter) getInboxRateLimit(ctx context.Context) domain.InboxRateLimit {
	var ghRate ghRateLimit
	if err := ghJSON(ctx, &ghRate, "api", "rate_limit"); err != nil {
		return domain.InboxRateLimit{}
	}
	return domain.InboxRateLimit{
		SearchLimit:      ghRate.Resources.Search.Limit,
		SearchRemaining:  ghRate.Resources.Search.Remaining,
		SearchResetAt:    unixReset(ghRate.Resources.Search.Reset),
		GraphQLLimit:     ghRate.Resources.GraphQL.Limit,
		GraphQLRemaining: ghRate.Resources.GraphQL.Remaining,
		GraphQLResetAt:   unixReset(ghRate.Resources.GraphQL.Reset),
	}
}

func unixReset(reset int) time.Time {
	if reset <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(reset), 0)
}

func toInboxItem(pr ghSearchPR, source domain.InboxSource, ci domain.CIStatus) (domain.InboxItem, bool) {
	repo, ok := parseNameWithOwner(pr.Repository.NameWithOwner)
	if !ok {
		return domain.InboxItem{}, false
	}
	if ci == "" {
		ci = domain.CINone
	}
	review := domain.ReviewStatus{State: domain.ReviewNone}
	if source == domain.InboxSourceAttention {
		review.State = domain.ReviewPending
	}
	return domain.InboxItem{
		PR: domain.PR{
			Number:         pr.Number,
			Title:          pr.Title,
			Author:         pr.Author.Login,
			State:          mapState(pr.State),
			Draft:          pr.IsDraft,
			CI:             ci,
			Review:         review,
			UpdatedAt:      pr.UpdatedAt,
			CreatedAt:      pr.CreatedAt,
			URL:            pr.URL,
			LastActivityAt: pr.UpdatedAt,
		},
		Repo:       repo,
		Sources:    []domain.InboxSource{source},
		LocalState: domain.InboxLocalAPIOnly,
	}, true
}

func parseNameWithOwner(nameWithOwner string) (domain.RepoRef, bool) {
	owner, name, ok := strings.Cut(nameWithOwner, "/")
	if !ok || owner == "" || name == "" {
		return domain.RepoRef{}, false
	}
	return domain.RepoRef{Owner: owner, Name: name}, true
}

func inboxItemKey(item domain.InboxItem) string {
	return fmt.Sprintf("%s#%d", item.Repo.String(), item.Number)
}

func mergeInboxSources(existing, incoming []domain.InboxSource) []domain.InboxSource {
	out := append([]domain.InboxSource(nil), existing...)
	for _, next := range incoming {
		seen := false
		for _, current := range out {
			if current == next {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, next)
		}
	}
	return out
}
