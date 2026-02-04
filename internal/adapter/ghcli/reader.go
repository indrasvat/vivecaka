package ghcli

import (
	"context"
	"fmt"
	"time"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// JSON field lists for gh pr list/view --json.
const (
	prListFields = "number,title,author,state,isDraft,headRefName,baseRefName,labels,statusCheckRollup,reviewDecision,updatedAt,createdAt,url"
	prViewFields = "number,title,author,state,isDraft,headRefName,baseRefName,labels,statusCheckRollup,reviewDecision,updatedAt,createdAt,url,body,assignees,reviewRequests,latestReviews,files"
	checkFields  = "name,status,conclusion,startedAt,completedAt,detailsUrl"
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

// ListPRs fetches PRs via gh pr list --json.
func (a *Adapter) ListPRs(ctx context.Context, repo domain.RepoRef, opts domain.ListOpts) ([]domain.PR, error) {
	args := []string{"pr", "list", "--json", prListFields}
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
	if opts.PerPage > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", opts.PerPage))
	}

	var ghPRs []ghPR
	if err := ghJSON(ctx, &ghPRs, args...); err != nil {
		return nil, fmt.Errorf("listing PRs: %w", err)
	}

	prs := make([]domain.PR, 0, len(ghPRs))
	for _, g := range ghPRs {
		pr := toDomainPR(g)
		// Client-side draft filter.
		switch opts.Draft {
		case domain.DraftExclude:
			if pr.Draft {
				continue
			}
		case domain.DraftOnly:
			if !pr.Draft {
				continue
			}
		}
		prs = append(prs, pr)
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

// GetComments fetches review comments for a PR via gh api.
func (a *Adapter) GetComments(ctx context.Context, repo domain.RepoRef, number int) ([]domain.CommentThread, error) {
	endpoint := fmt.Sprintf("repos/%s/pulls/%d/comments", repo, number)
	args := []string{"api", endpoint, "--paginate"}

	var ghComments []ghAPIComment
	if err := ghJSON(ctx, &ghComments, args...); err != nil {
		return nil, fmt.Errorf("getting comments for PR #%d: %w", number, err)
	}

	return groupCommentsIntoThreads(ghComments), nil
}

// ghAPIComment is the shape of a review comment from the REST API.
type ghAPIComment struct {
	ID                  int       `json:"id"`
	NodeID              string    `json:"node_id"`
	Path                string    `json:"path"`
	Line                *int      `json:"line"`
	OriginalLine        *int      `json:"original_line"`
	Body                string    `json:"body"`
	User                ghActor   `json:"user"`
	CreatedAt           time.Time `json:"created_at"`
	InReplyToID         *int      `json:"in_reply_to_id"`
	PullRequestReviewID int       `json:"pull_request_review_id"`
}

// groupCommentsIntoThreads groups flat comments into threads by in_reply_to_id.
func groupCommentsIntoThreads(comments []ghAPIComment) []domain.CommentThread {
	threadMap := make(map[int]*domain.CommentThread)
	var threadOrder []int

	for _, c := range comments {
		comment := domain.Comment{
			ID:        fmt.Sprintf("%d", c.ID),
			Author:    c.User.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		}

		if c.InReplyToID != nil {
			// Reply to existing thread.
			if thread, ok := threadMap[*c.InReplyToID]; ok {
				thread.Comments = append(thread.Comments, comment)
				continue
			}
		}

		// New thread root.
		line := 0
		if c.Line != nil {
			line = *c.Line
		}
		thread := domain.CommentThread{
			ID:       fmt.Sprintf("%d", c.ID),
			Path:     c.Path,
			Line:     line,
			Comments: []domain.Comment{comment},
		}
		threadMap[c.ID] = &thread
		threadOrder = append(threadOrder, c.ID)
	}

	threads := make([]domain.CommentThread, 0, len(threadOrder))
	for _, id := range threadOrder {
		threads = append(threads, *threadMap[id])
	}
	return threads
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
