package review

import (
	"context"
	"fmt"
	"sort"

	"github.com/shurcooL/githubv4"

	"github.com/ivanov-gv/gh-contribute/internal/utils/format"
)

// Service provides review detail operations via GraphQL
type Service struct {
	gql   *githubv4.Client
	owner string
	repo  string
}

// NewService creates a new review service
func NewService(gql *githubv4.Client, owner, repo string) *Service {
	return &Service{gql: gql, owner: owner, repo: repo}
}

// ReviewComment holds a single comment that belongs to the requested review
type ReviewComment struct {
	DatabaseID        int64
	Author            string
	Body              string
	CreatedAt         string
	ReplyToID         int64 // 0 if thread root
	ReplyToIsExternal bool  // true when ReplyToID refers to a comment outside this review
	IsMinimized       bool
	MinimizedReason   string
	Reactions         []format.Reaction
}

// ReviewThreadGroup holds the comments from this review that belong to the same thread.
// ThreadID is the database ID of the first comment in the full thread — use it with
// the `thread` command to see the complete thread including cross-review comments.
type ReviewThreadGroup struct {
	ThreadID          int64
	IsOutdated        bool
	Path              string
	Line              int
	StartLine         int
	OriginalLine      int
	OriginalStartLine int
	DiffHunk          string // populated only when showDiff is true
	Comments          []ReviewComment
}

// ReviewDetail holds the full review with its thread groups
type ReviewDetail struct {
	DatabaseID   int64
	Author       string
	Body         string
	State        string
	CreatedAt    string
	ViewerLogin  string
	Reactions    []format.Reaction
	ThreadGroups []ReviewThreadGroup
}

// reactionNode is a single reaction with content and author
type reactionNode struct {
	Content githubv4.String
	User    struct {
		Login githubv4.String
	}
}

// reviewMetaNode holds review-level metadata (no inline comments)
type reviewMetaNode struct {
	DatabaseID int64
	Author     struct {
		Login githubv4.String
	}
	Body      githubv4.String
	State     githubv4.String
	CreatedAt githubv4.DateTime
	Reactions struct {
		Nodes []reactionNode
	} `graphql:"reactions(first: 20)"`
}

// threadCommentNodeNoDiff — comment within a review thread, without diffHunk
type threadCommentNodeNoDiff struct {
	DatabaseID int64
	Author     struct {
		Login githubv4.String
	}
	Body            githubv4.String
	CreatedAt       githubv4.DateTime
	IsMinimized     githubv4.Boolean
	MinimizedReason githubv4.String
	ReplyTo         *struct {
		DatabaseID int64
	}
	PullRequestReview *struct {
		DatabaseID int64
	}
	Reactions struct {
		Nodes []reactionNode
	} `graphql:"reactions(first: 20)"`
}

// threadCommentNodeWithDiff — comment within a review thread, with diffHunk
type threadCommentNodeWithDiff struct {
	DatabaseID int64
	Author     struct {
		Login githubv4.String
	}
	Body            githubv4.String
	CreatedAt       githubv4.DateTime
	IsMinimized     githubv4.Boolean
	MinimizedReason githubv4.String
	DiffHunk        githubv4.String
	ReplyTo         *struct {
		DatabaseID int64
	}
	PullRequestReview *struct {
		DatabaseID int64
	}
	Reactions struct {
		Nodes []reactionNode
	} `graphql:"reactions(first: 20)"`
}

// reviewThreadNodeNoDiff — a review thread without diffHunk on comments
type reviewThreadNodeNoDiff struct {
	IsOutdated        githubv4.Boolean
	Path              githubv4.String
	Line              *githubv4.Int
	StartLine         *githubv4.Int
	OriginalLine      *githubv4.Int
	OriginalStartLine *githubv4.Int
	Comments          struct {
		Nodes []threadCommentNodeNoDiff
	} `graphql:"comments(first: 50)"`
}

// reviewThreadNodeWithDiff — a review thread with diffHunk on comments
type reviewThreadNodeWithDiff struct {
	IsOutdated        githubv4.Boolean
	Path              githubv4.String
	Line              *githubv4.Int
	StartLine         *githubv4.Int
	OriginalLine      *githubv4.Int
	OriginalStartLine *githubv4.Int
	Comments          struct {
		Nodes []threadCommentNodeWithDiff
	} `graphql:"comments(first: 50)"`
}

// allReviewsQueryNoDiff — review metadata + all threads without diffHunk
type allReviewsQueryNoDiff struct {
	Viewer struct {
		Login githubv4.String
	}
	Repository struct {
		PullRequest struct {
			Reviews struct {
				Nodes []reviewMetaNode
			} `graphql:"reviews(first: 100)"`
			ReviewThreads struct {
				Nodes []reviewThreadNodeNoDiff
			} `graphql:"reviewThreads(first: 100)"`
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// allReviewsQueryWithDiff — review metadata + all threads with diffHunk
type allReviewsQueryWithDiff struct {
	Viewer struct {
		Login githubv4.String
	}
	Repository struct {
		PullRequest struct {
			Reviews struct {
				Nodes []reviewMetaNode
			} `graphql:"reviews(first: 100)"`
			ReviewThreads struct {
				Nodes []reviewThreadNodeWithDiff
			} `graphql:"reviewThreads(first: 100)"`
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// Get returns the review detail containing only comments that belong to this review,
// grouped into thread groups. Each group carries the ThreadID for full-thread lookup.
// When showDiff is true, diffHunk is included (taken from the first thread comment).
func (s *Service) Get(prNumber int, reviewDatabaseID int64, showDiff bool) (*ReviewDetail, error) {
	variables := map[string]interface{}{
		"owner":  githubv4.String(s.owner),
		"repo":   githubv4.String(s.repo),
		"number": githubv4.Int(prNumber),
	}

	if showDiff {
		var query allReviewsQueryWithDiff
		if err := s.gql.Query(context.Background(), &query, variables); err != nil {
			return nil, fmt.Errorf("gql.Query [pr=%d, review=%d]: %w", prNumber, reviewDatabaseID, err)
		}
		meta := findReviewMeta(query.Repository.PullRequest.Reviews.Nodes, reviewDatabaseID)
		if meta == nil {
			return nil, fmt.Errorf("review #%d not found in PR #%d", reviewDatabaseID, prNumber)
		}
		groups := collectGroupsWithDiff(query.Repository.PullRequest.ReviewThreads.Nodes, reviewDatabaseID)
		return buildReviewDetail(meta, string(query.Viewer.Login), groups), nil
	}

	var query allReviewsQueryNoDiff
	if err := s.gql.Query(context.Background(), &query, variables); err != nil {
		return nil, fmt.Errorf("gql.Query [pr=%d, review=%d]: %w", prNumber, reviewDatabaseID, err)
	}
	meta := findReviewMeta(query.Repository.PullRequest.Reviews.Nodes, reviewDatabaseID)
	if meta == nil {
		return nil, fmt.Errorf("review #%d not found in PR #%d", reviewDatabaseID, prNumber)
	}
	groups := collectGroupsNoDiff(query.Repository.PullRequest.ReviewThreads.Nodes, reviewDatabaseID)
	return buildReviewDetail(meta, string(query.Viewer.Login), groups), nil
}

func findReviewMeta(nodes []reviewMetaNode, reviewDatabaseID int64) *reviewMetaNode {
	for i := range nodes {
		if nodes[i].DatabaseID == reviewDatabaseID {
			return &nodes[i]
		}
	}
	return nil
}

// collectGroupsNoDiff builds thread groups from the no-diff thread nodes.
func collectGroupsNoDiff(nodes []reviewThreadNodeNoDiff, reviewDatabaseID int64) []ReviewThreadGroup {
	var groups []ReviewThreadGroup
	for _, n := range nodes {
		group, ok := buildGroupNoDiff(n, reviewDatabaseID)
		if ok {
			groups = append(groups, group)
		}
	}
	return sortedGroups(groups)
}

// collectGroupsWithDiff builds thread groups from the with-diff thread nodes.
func collectGroupsWithDiff(nodes []reviewThreadNodeWithDiff, reviewDatabaseID int64) []ReviewThreadGroup {
	var groups []ReviewThreadGroup
	for _, n := range nodes {
		group, ok := buildGroupWithDiff(n, reviewDatabaseID)
		if ok {
			groups = append(groups, group)
		}
	}
	return sortedGroups(groups)
}

func buildGroupNoDiff(n reviewThreadNodeNoDiff, reviewDatabaseID int64) (ReviewThreadGroup, bool) {
	// collect only comments from this review, preserving thread order
	var reviewComments []threadCommentNodeNoDiff
	for _, c := range n.Comments.Nodes {
		if c.PullRequestReview != nil && c.PullRequestReview.DatabaseID == reviewDatabaseID {
			reviewComments = append(reviewComments, c)
		}
	}
	if len(reviewComments) == 0 {
		return ReviewThreadGroup{}, false
	}

	group := newThreadGroup(n.IsOutdated, n.Path, n.Line, n.StartLine, n.OriginalLine, n.OriginalStartLine, n.Comments.Nodes)
	// build set of this review's comment IDs for external-reply detection
	reviewIDs := commentIDSet(reviewComments)
	for _, c := range reviewComments {
		rc := mapReviewComment(c.DatabaseID, c.Author.Login, c.Body, c.CreatedAt,
			c.IsMinimized, c.MinimizedReason, c.ReplyTo, c.Reactions.Nodes, reviewIDs)
		group.Comments = append(group.Comments, rc)
	}
	return group, true
}

func buildGroupWithDiff(n reviewThreadNodeWithDiff, reviewDatabaseID int64) (ReviewThreadGroup, bool) {
	var reviewComments []threadCommentNodeWithDiff
	for _, c := range n.Comments.Nodes {
		if c.PullRequestReview != nil && c.PullRequestReview.DatabaseID == reviewDatabaseID {
			reviewComments = append(reviewComments, c)
		}
	}
	if len(reviewComments) == 0 {
		return ReviewThreadGroup{}, false
	}

	group := newThreadGroup(n.IsOutdated, n.Path, n.Line, n.StartLine, n.OriginalLine, n.OriginalStartLine, n.Comments.Nodes)
	// diffHunk is the same for all comments in a thread — take from first
	if len(n.Comments.Nodes) > 0 {
		group.DiffHunk = string(n.Comments.Nodes[0].DiffHunk)
	}
	reviewIDs := commentIDSet(reviewComments)
	for _, c := range reviewComments {
		rc := mapReviewComment(c.DatabaseID, c.Author.Login, c.Body, c.CreatedAt,
			c.IsMinimized, c.MinimizedReason, c.ReplyTo, c.Reactions.Nodes, reviewIDs)
		group.Comments = append(group.Comments, rc)
	}
	return group, true
}

// newThreadGroup initialises a ReviewThreadGroup with location info and ThreadID.
// ThreadID is the database ID of the first comment in the full thread.
func newThreadGroup[C interface{ getID() int64 }](
	isOutdated githubv4.Boolean,
	path githubv4.String,
	line, startLine, originalLine, originalStartLine *githubv4.Int,
	allComments []C,
) ReviewThreadGroup {
	g := ReviewThreadGroup{
		IsOutdated: bool(isOutdated),
		Path:       string(path),
	}
	if len(allComments) > 0 {
		g.ThreadID = allComments[0].getID()
	}
	if line != nil {
		g.Line = int(*line)
	}
	if startLine != nil {
		g.StartLine = int(*startLine)
	}
	if originalLine != nil {
		g.OriginalLine = int(*originalLine)
	}
	if originalStartLine != nil {
		g.OriginalStartLine = int(*originalStartLine)
	}
	return g
}

func (c threadCommentNodeNoDiff) getID() int64   { return c.DatabaseID }
func (c threadCommentNodeWithDiff) getID() int64 { return c.DatabaseID }

// commentIDSet builds a set of database IDs for fast membership checks.
func commentIDSet[C interface{ getID() int64 }](comments []C) map[int64]struct{} {
	ids := make(map[int64]struct{}, len(comments))
	for _, c := range comments {
		ids[c.getID()] = struct{}{}
	}
	return ids
}

func mapReviewComment(
	databaseID int64,
	authorLogin githubv4.String,
	body githubv4.String,
	createdAt githubv4.DateTime,
	isMinimized githubv4.Boolean,
	minimizedReason githubv4.String,
	replyTo *struct{ DatabaseID int64 },
	reactions []reactionNode,
	reviewIDs map[int64]struct{},
) ReviewComment {
	rc := ReviewComment{
		DatabaseID:      databaseID,
		Author:          string(authorLogin),
		Body:            string(body),
		CreatedAt:       createdAt.UTC().Format("2006-01-02T15:04:05Z"),
		IsMinimized:     bool(isMinimized),
		MinimizedReason: string(minimizedReason),
		Reactions:       mapReactions(reactions),
	}
	if replyTo != nil {
		rc.ReplyToID = replyTo.DatabaseID
		_, inReview := reviewIDs[replyTo.DatabaseID]
		rc.ReplyToIsExternal = !inReview
	}
	return rc
}

func buildReviewDetail(n *reviewMetaNode, viewerLogin string, groups []ReviewThreadGroup) *ReviewDetail {
	return &ReviewDetail{
		DatabaseID:   n.DatabaseID,
		Author:       string(n.Author.Login),
		Body:         string(n.Body),
		State:        string(n.State),
		CreatedAt:    n.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		ViewerLogin:  viewerLogin,
		Reactions:    mapReactions(n.Reactions.Nodes),
		ThreadGroups: groups,
	}
}

// sortedGroups sorts thread groups by the creation time of their first comment.
func sortedGroups(groups []ReviewThreadGroup) []ReviewThreadGroup {
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Comments) == 0 || len(groups[j].Comments) == 0 {
			return false
		}
		return groups[i].Comments[0].CreatedAt < groups[j].Comments[0].CreatedAt
	})
	return groups
}

func mapReactions(nodes []reactionNode) []format.Reaction {
	reactions := make([]format.Reaction, len(nodes))
	for i, n := range nodes {
		reactions[i] = format.Reaction{Content: string(n.Content), Author: string(n.User.Login)}
	}
	return reactions
}
