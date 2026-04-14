package ghcli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// JSON field lists for gh pr list/view --json.
const (
	// prListFields is for the initial load (single page). Includes statusCheckRollup for CI status.
	prListFields = "number,title,author,state,isDraft,headRefName,baseRefName,labels,statusCheckRollup,reviewDecision,updatedAt,createdAt,url"
	// prListFieldsLight is for pagination (loading more PRs). Excludes statusCheckRollup to avoid API timeouts.
	// CI status will show as "none" for paginated items until detail view is opened.
	prListFieldsLight = "number,title,author,state,isDraft,headRefName,baseRefName,labels,reviewDecision,updatedAt,createdAt,url"
	prViewFields      = "number,title,author,state,isDraft,headRefName,baseRefName,labels,statusCheckRollup,reviewDecision,updatedAt,createdAt,url,body,assignees,reviewRequests,latestReviews,files"
	checkFields       = "name,status,conclusion,startedAt,completedAt,detailsUrl"
)

// ghPR is the JSON shape returned by gh pr list/view.
type ghPR struct {
	Number            int           `json:"number"`
	Title             string        `json:"title"`
	Author            ghActor       `json:"author"`
	State             string        `json:"state"`
	IsDraft           bool          `json:"isDraft"`
	HeadRefName       string        `json:"headRefName"`
	BaseRefName       string        `json:"baseRefName"`
	Labels            []ghLabel     `json:"labels"`
	StatusCheckRollup []ghCheck     `json:"statusCheckRollup"`
	ReviewDecision    string        `json:"reviewDecision"`
	UpdatedAt         time.Time     `json:"updatedAt"`
	CreatedAt         time.Time     `json:"createdAt"`
	URL               string        `json:"url"`
	Body              string        `json:"body"`
	Assignees         []ghActor     `json:"assignees"`
	ReviewRequests    []ghReviewReq `json:"reviewRequests"`
	LatestReviews     []ghReview    `json:"latestReviews"`
	Files             []ghFile      `json:"files"`
}

type ghActor struct {
	Login string `json:"login"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghCheck struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Conclusion  string     `json:"conclusion"`
	StartedAt   *time.Time `json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	DetailsURL  string     `json:"detailsUrl"`
}

type ghReviewReq struct {
	Login string `json:"login"`
	// For team reviews this is under "name" key.
	Name string `json:"name"`
}

type ghReview struct {
	Author ghActor `json:"author"`
	State  string  `json:"state"`
}

type ghFile struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// GetPRCount fetches the total number of open PRs for a repo via GraphQL.
func (a *Adapter) GetPRCount(ctx context.Context, repo domain.RepoRef, state domain.PRState) (int, error) {
	// Map state to GraphQL enum
	gqlState := "OPEN"
	switch state {
	case domain.PRStateClosed:
		gqlState = "CLOSED"
	case domain.PRStateMerged:
		gqlState = "MERGED"
	}

	query := fmt.Sprintf(`query { repository(owner: %q, name: %q) { pullRequests(states: %s) { totalCount } } }`,
		repo.Owner, repo.Name, gqlState)

	args := []string{"api", "graphql", "-f", "query=" + query}

	var result struct {
		Data struct {
			Repository struct {
				PullRequests struct {
					TotalCount int `json:"totalCount"`
				} `json:"pullRequests"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := ghJSON(ctx, &result, args...); err != nil {
		return 0, fmt.Errorf("getting PR count: %w", err)
	}

	return result.Data.Repository.PullRequests.TotalCount, nil
}

// ListPRs fetches PRs via gh pr list --json.
// Supports pagination via opts.Page and opts.PerPage.
// Page is 1-based. For page > 1, we fetch page*perPage items and return only items for that page.
// Note: For pagination (page > 1), we use a lighter field list that excludes statusCheckRollup
// to avoid GitHub API timeouts on large result sets.
func (a *Adapter) ListPRs(ctx context.Context, repo domain.RepoRef, opts domain.ListOpts) ([]domain.PR, error) {
	// Use light fields for pagination to avoid API timeouts
	fields := prListFields
	if opts.Page > 1 {
		fields = prListFieldsLight
	}
	args := []string{"pr", "list", "--json", fields}
	args = append(args, repoArgs(repo)...)

	if opts.State != "" && opts.State != "all" {
		args = append(args, "--state", string(opts.State))
	}
	if opts.Author != "" {
		args = append(args, "--author", opts.Author)
	}
	for _, l := range opts.Labels {
		args = append(args, "--label", l)
	}
	if opts.Search != "" {
		args = append(args, "--search", opts.Search)
	}

	// Calculate limit for pagination
	// For page N, we need to fetch N*PerPage items total and skip the first (N-1)*PerPage
	page := opts.Page
	page = max(page, 1)
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 50 // default
	}
	limit := page * perPage
	args = append(args, "--limit", fmt.Sprintf("%d", limit))

	var ghPRs []ghPR
	if err := ghJSON(ctx, &ghPRs, args...); err != nil {
		return nil, fmt.Errorf("listing PRs: %w", err)
	}

	// Apply client-side draft filter BEFORE pagination so that excluded
	// drafts don't reduce the effective page size.
	filtered := ghPRs[:0:0]
	for _, g := range ghPRs {
		switch opts.Draft {
		case domain.DraftExclude:
			if g.IsDraft {
				continue
			}
		case domain.DraftOnly:
			if !g.IsDraft {
				continue
			}
		}
		filtered = append(filtered, g)
	}

	// Paginate over the filtered set.
	startIdx := (page - 1) * perPage
	if startIdx >= len(filtered) {
		return []domain.PR{}, nil
	}
	pageItems := filtered[startIdx:]

	prs := make([]domain.PR, 0, len(pageItems))
	for _, g := range pageItems {
		prs = append(prs, toDomainPR(g))
	}

	return prs, nil
}

// GetPR fetches a single PR with full details via gh pr view --json.
func (a *Adapter) GetPR(ctx context.Context, repo domain.RepoRef, number int) (*domain.PRDetail, error) {
	args := []string{"pr", "view", fmt.Sprintf("%d", number), "--json", prViewFields}
	args = append(args, repoArgs(repo)...)

	var g ghPR
	if err := ghJSON(ctx, &g, args...); err != nil {
		return nil, fmt.Errorf("getting PR #%d: %w", number, err)
	}

	detail := toDomainPRDetail(g)
	return &detail, nil
}

// GetDiff fetches the raw diff for a PR and parses it.
func (a *Adapter) GetDiff(ctx context.Context, repo domain.RepoRef, number int) (*domain.Diff, error) {
	args := []string{"pr", "diff", fmt.Sprintf("%d", number)}
	args = append(args, repoArgs(repo)...)

	out, err := ghExec(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("getting diff for PR #%d: %w", number, err)
	}

	diff := ParseDiff(string(out))
	return &diff, nil
}

// GetChecks fetches CI check results for a PR.
func (a *Adapter) GetChecks(ctx context.Context, repo domain.RepoRef, number int) ([]domain.Check, error) {
	args := []string{"pr", "checks", fmt.Sprintf("%d", number), "--json", checkFields}
	args = append(args, repoArgs(repo)...)

	var ghChecks []ghCheck
	if err := ghJSON(ctx, &ghChecks, args...); err != nil {
		return nil, fmt.Errorf("getting checks for PR #%d: %w", number, err)
	}

	checks := make([]domain.Check, 0, len(ghChecks))
	for _, c := range ghChecks {
		checks = append(checks, toDomainCheck(c))
	}
	return checks, nil
}

// GetComments fetches inline review threads for a PR via GraphQL.
func (a *Adapter) GetComments(ctx context.Context, repo domain.RepoRef, number int) ([]domain.CommentThread, error) {
	var (
		cursor  string
		threads []domain.CommentThread
	)

	for {
		page, err := fetchReviewThreadsPage(ctx, repo, number, cursor)
		if err != nil {
			return nil, fmt.Errorf("getting comments for PR #%d: %w", number, err)
		}
		pageThreads, err := expandReviewThreadComments(ctx, page.Nodes, fetchReviewThreadCommentsPage)
		if err != nil {
			return nil, fmt.Errorf("getting comments for PR #%d: %w", number, err)
		}
		threads = append(threads, toDomainCommentThreads(pageThreads)...)
		if !page.PageInfo.HasNextPage {
			break
		}
		cursor = page.PageInfo.EndCursor
	}

	return threads, nil
}

// GetDiscussion fetches non-inline PR discussion items (review bodies + top-level PR comments).
func (a *Adapter) GetDiscussion(ctx context.Context, repo domain.RepoRef, number int) ([]domain.DiscussionItem, error) {
	var discussion []domain.DiscussionItem

	commentCursor := ""
	for {
		page, err := fetchIssueCommentsPage(ctx, repo, number, commentCursor)
		if err != nil {
			return nil, fmt.Errorf("getting PR conversation comments for PR #%d: %w", number, err)
		}
		discussion = append(discussion, toDomainIssueComments(page.Nodes)...)
		if !page.PageInfo.HasNextPage {
			break
		}
		commentCursor = page.PageInfo.EndCursor
	}

	reviewCursor := ""
	for {
		page, err := fetchReviewsPage(ctx, repo, number, reviewCursor)
		if err != nil {
			return nil, fmt.Errorf("getting reviews for PR #%d: %w", number, err)
		}
		discussion = append(discussion, toDomainReviewItems(page.Nodes)...)
		if !page.PageInfo.HasNextPage {
			break
		}
		reviewCursor = page.PageInfo.EndCursor
	}

	return discussion, nil
}

type ghPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type ghGraphQLComment struct {
	ID         string    `json:"id"`
	DatabaseID int       `json:"databaseId"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
	URL        string    `json:"url"`
	Author     ghActor   `json:"author"`
}

type ghGraphQLCommentConnection struct {
	Nodes    []ghGraphQLComment `json:"nodes"`
	PageInfo ghPageInfo         `json:"pageInfo"`
}

type ghReviewThread struct {
	ID         string                     `json:"id"`
	IsResolved bool                       `json:"isResolved"`
	Path       string                     `json:"path"`
	Line       *int                       `json:"line"`
	Comments   ghGraphQLCommentConnection `json:"comments"`
}

type ghReviewThreadsPage struct {
	Nodes    []ghReviewThread `json:"nodes"`
	PageInfo ghPageInfo       `json:"pageInfo"`
}

type ghReviewSummary struct {
	ID          string    `json:"id"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submittedAt"`
	URL         string    `json:"url"`
	Author      ghActor   `json:"author"`
}

type ghReviewsPage struct {
	Nodes    []ghReviewSummary `json:"nodes"`
	PageInfo ghPageInfo        `json:"pageInfo"`
}

type ghIssueComment struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url"`
	Author    ghActor   `json:"author"`
}

type ghIssueCommentsPage struct {
	Nodes    []ghIssueComment `json:"nodes"`
	PageInfo ghPageInfo       `json:"pageInfo"`
}

func fetchReviewThreadsPage(ctx context.Context, repo domain.RepoRef, number int, cursor string) (*ghReviewThreadsPage, error) {
	after := "null"
	if cursor != "" {
		after = fmt.Sprintf("%q", cursor)
	}

	query := fmt.Sprintf(`query {
  repository(owner: %q, name: %q) {
    pullRequest(number: %d) {
      reviewThreads(first: 100, after: %s) {
        nodes {
          id
          isResolved
          path
          line
	          comments(first: 100) {
	            nodes {
	              id
	              databaseId
	              body
	              createdAt
	              url
	              author { login }
	            }
	            pageInfo { hasNextPage endCursor }
	          }
	        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}`, repo.Owner, repo.Name, number, after)

	var result struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads ghReviewThreadsPage `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := ghJSON(ctx, &result, "api", "graphql", "-f", "query="+query); err != nil {
		return nil, err
	}
	return &result.Data.Repository.PullRequest.ReviewThreads, nil
}

func fetchReviewThreadCommentsPage(ctx context.Context, threadID, cursor string) (*ghGraphQLCommentConnection, error) {
	after := "null"
	if cursor != "" {
		after = fmt.Sprintf("%q", cursor)
	}

	query := fmt.Sprintf(`query {
  node(id: %q) {
    ... on PullRequestReviewThread {
      comments(first: 100, after: %s) {
        nodes {
          id
          databaseId
          body
          createdAt
          url
          author { login }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}`, threadID, after)

	var result struct {
		Data struct {
			Node struct {
				Comments ghGraphQLCommentConnection `json:"comments"`
			} `json:"node"`
		} `json:"data"`
	}
	if err := ghJSON(ctx, &result, "api", "graphql", "-f", "query="+query); err != nil {
		return nil, err
	}
	return &result.Data.Node.Comments, nil
}

func expandReviewThreadComments(
	ctx context.Context,
	threads []ghReviewThread,
	fetchPage func(context.Context, string, string) (*ghGraphQLCommentConnection, error),
) ([]ghReviewThread, error) {
	out := make([]ghReviewThread, len(threads))
	copy(out, threads)

	for i := range out {
		commentPage := out[i].Comments
		if !commentPage.PageInfo.HasNextPage {
			continue
		}

		allComments := append([]ghGraphQLComment(nil), commentPage.Nodes...)
		cursor := commentPage.PageInfo.EndCursor

		for commentPage.PageInfo.HasNextPage {
			page, err := fetchPage(ctx, out[i].ID, cursor)
			if err != nil {
				return nil, err
			}
			allComments = append(allComments, page.Nodes...)
			commentPage = *page
			cursor = commentPage.PageInfo.EndCursor
		}

		out[i].Comments = ghGraphQLCommentConnection{Nodes: allComments}
	}

	return out, nil
}

func fetchReviewsPage(ctx context.Context, repo domain.RepoRef, number int, cursor string) (*ghReviewsPage, error) {
	after := "null"
	if cursor != "" {
		after = fmt.Sprintf("%q", cursor)
	}

	query := fmt.Sprintf(`query {
  repository(owner: %q, name: %q) {
    pullRequest(number: %d) {
      reviews(first: 100, after: %s) {
        nodes {
          id
          body
          state
          submittedAt
          url
          author { login }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}`, repo.Owner, repo.Name, number, after)

	var result struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					Reviews ghReviewsPage `json:"reviews"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := ghJSON(ctx, &result, "api", "graphql", "-f", "query="+query); err != nil {
		return nil, err
	}
	return &result.Data.Repository.PullRequest.Reviews, nil
}

func fetchIssueCommentsPage(ctx context.Context, repo domain.RepoRef, number int, cursor string) (*ghIssueCommentsPage, error) {
	after := "null"
	if cursor != "" {
		after = fmt.Sprintf("%q", cursor)
	}

	query := fmt.Sprintf(`query {
  repository(owner: %q, name: %q) {
    pullRequest(number: %d) {
      comments(first: 100, after: %s) {
        nodes {
          id
          body
          createdAt
          url
          author { login }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}`, repo.Owner, repo.Name, number, after)

	var result struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					Comments ghIssueCommentsPage `json:"comments"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := ghJSON(ctx, &result, "api", "graphql", "-f", "query="+query); err != nil {
		return nil, err
	}
	return &result.Data.Repository.PullRequest.Comments, nil
}

func toDomainCommentThreads(threads []ghReviewThread) []domain.CommentThread {
	out := make([]domain.CommentThread, 0, len(threads))
	for _, thread := range threads {
		if len(thread.Comments.Nodes) == 0 {
			continue
		}
		line := 0
		if thread.Line != nil {
			line = *thread.Line
		}

		comments := make([]domain.Comment, 0, len(thread.Comments.Nodes))
		replyToID := ""
		rootID := ""
		for i, c := range thread.Comments.Nodes {
			if i == 0 {
				rootID = fmt.Sprintf("%d", c.DatabaseID)
				replyToID = rootID
			}
			comments = append(comments, domain.Comment{
				ID:        fmt.Sprintf("%d", c.DatabaseID),
				Author:    c.Author.Login,
				Body:      c.Body,
				CreatedAt: c.CreatedAt,
				URL:       c.URL,
			})
		}

		out = append(out, domain.CommentThread{
			ID:        rootID,
			ThreadID:  thread.ID,
			ReplyToID: replyToID,
			Path:      thread.Path,
			Line:      line,
			Resolved:  thread.IsResolved,
			Comments:  comments,
		})
	}
	return out
}

func toDomainReviewItems(reviews []ghReviewSummary) []domain.DiscussionItem {
	items := make([]domain.DiscussionItem, 0, len(reviews))
	for _, review := range reviews {
		if strings.TrimSpace(review.Body) == "" {
			continue
		}
		items = append(items, domain.DiscussionItem{
			ID:          review.ID,
			Kind:        domain.DiscussionReview,
			ReviewState: mapReviewState(review.State),
			StateLabel:  strings.ToLower(strings.ReplaceAll(review.State, "_", " ")),
			CreatedAt:   review.SubmittedAt,
			URL:         review.URL,
			Comments: []domain.Comment{{
				ID:        review.ID,
				Author:    review.Author.Login,
				Body:      review.Body,
				CreatedAt: review.SubmittedAt,
				URL:       review.URL,
			}},
		})
	}
	return items
}

func toDomainIssueComments(comments []ghIssueComment) []domain.DiscussionItem {
	items := make([]domain.DiscussionItem, 0, len(comments))
	for _, comment := range comments {
		if strings.TrimSpace(comment.Body) == "" {
			continue
		}
		items = append(items, domain.DiscussionItem{
			ID:        comment.ID,
			Kind:      domain.DiscussionComment,
			CreatedAt: comment.CreatedAt,
			URL:       comment.URL,
			Comments: []domain.Comment{{
				ID:        comment.ID,
				Author:    comment.Author.Login,
				Body:      comment.Body,
				CreatedAt: comment.CreatedAt,
				URL:       comment.URL,
			}},
		})
	}
	return items
}

// toDomainPR converts a ghPR to a domain.PR.
func toDomainPR(g ghPR) domain.PR {
	labels := make([]string, len(g.Labels))
	for i, l := range g.Labels {
		labels[i] = l.Name
	}

	return domain.PR{
		Number: g.Number,
		Title:  g.Title,
		Author: g.Author.Login,
		State:  mapState(g.State),
		Draft:  g.IsDraft,
		Branch: domain.BranchInfo{
			Head: g.HeadRefName,
			Base: g.BaseRefName,
		},
		Labels:         labels,
		CI:             aggregateCI(g.StatusCheckRollup),
		Review:         mapReviewDecision(g.ReviewDecision),
		UpdatedAt:      g.UpdatedAt,
		CreatedAt:      g.CreatedAt,
		URL:            g.URL,
		LastActivityAt: g.UpdatedAt,
	}
}

// toDomainPRDetail converts a ghPR to a domain.PRDetail.
func toDomainPRDetail(g ghPR) domain.PRDetail {
	assignees := make([]string, len(g.Assignees))
	for i, a := range g.Assignees {
		assignees[i] = a.Login
	}

	reviewers := make([]domain.ReviewerInfo, 0)
	// Add review requests (pending).
	for _, rr := range g.ReviewRequests {
		login := rr.Login
		if login == "" {
			login = rr.Name
		}
		reviewers = append(reviewers, domain.ReviewerInfo{
			Login: login,
			State: domain.ReviewPending,
		})
	}
	// Add completed reviews.
	for _, r := range g.LatestReviews {
		reviewers = append(reviewers, domain.ReviewerInfo{
			Login: r.Author.Login,
			State: mapReviewState(r.State),
		})
	}

	files := make([]domain.FileChange, len(g.Files))
	for i, f := range g.Files {
		files[i] = domain.FileChange{
			Path:      f.Path,
			Additions: f.Additions,
			Deletions: f.Deletions,
			Status:    "modified",
		}
	}

	checks := make([]domain.Check, len(g.StatusCheckRollup))
	for i, c := range g.StatusCheckRollup {
		checks[i] = toDomainCheck(c)
	}

	return domain.PRDetail{
		PR:        toDomainPR(g),
		Body:      g.Body,
		Assignees: assignees,
		Reviewers: reviewers,
		Checks:    checks,
		Files:     files,
	}
}

func toDomainCheck(c ghCheck) domain.Check {
	var duration time.Duration
	if c.StartedAt != nil && c.CompletedAt != nil {
		duration = c.CompletedAt.Sub(*c.StartedAt)
	}
	return domain.Check{
		Name:     c.Name,
		Status:   mapCheckStatus(c.Status, c.Conclusion),
		Duration: duration,
		URL:      c.DetailsURL,
	}
}

func mapState(s string) domain.PRState {
	switch s {
	case "OPEN":
		return domain.PRStateOpen
	case "CLOSED":
		return domain.PRStateClosed
	case "MERGED":
		return domain.PRStateMerged
	default:
		return domain.PRStateOpen
	}
}

func aggregateCI(checks []ghCheck) domain.CIStatus {
	if len(checks) == 0 {
		return domain.CINone
	}
	hasPending := false
	hasFail := false
	for _, c := range checks {
		st := mapCheckStatus(c.Status, c.Conclusion)
		switch st {
		case domain.CIFail:
			hasFail = true
		case domain.CIPending:
			hasPending = true
		}
	}
	if hasFail {
		return domain.CIFail
	}
	if hasPending {
		return domain.CIPending
	}
	return domain.CIPass
}

func mapCheckStatus(status, conclusion string) domain.CIStatus {
	switch status {
	case "COMPLETED":
		switch conclusion {
		case "SUCCESS":
			return domain.CIPass
		case "FAILURE", "TIMED_OUT", "STARTUP_FAILURE":
			return domain.CIFail
		case "SKIPPED", "NEUTRAL":
			return domain.CISkipped
		default:
			return domain.CIFail
		}
	case "IN_PROGRESS", "QUEUED", "PENDING", "WAITING", "REQUESTED":
		return domain.CIPending
	default:
		return domain.CINone
	}
}

func mapReviewDecision(decision string) domain.ReviewStatus {
	switch decision {
	case "APPROVED":
		return domain.ReviewStatus{State: domain.ReviewApproved}
	case "CHANGES_REQUESTED":
		return domain.ReviewStatus{State: domain.ReviewChangesRequested}
	case "REVIEW_REQUIRED":
		return domain.ReviewStatus{State: domain.ReviewPending}
	default:
		return domain.ReviewStatus{State: domain.ReviewNone}
	}
}

func mapReviewState(state string) domain.ReviewState {
	switch state {
	case "APPROVED":
		return domain.ReviewApproved
	case "CHANGES_REQUESTED":
		return domain.ReviewChangesRequested
	case "PENDING", "COMMENTED":
		return domain.ReviewPending
	default:
		return domain.ReviewNone
	}
}
