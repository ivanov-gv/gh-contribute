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

// ReviewComment holds a single inline review comment
// Note: CommitSHA/OriginalCommitSHA are intentionally absent — commit fields require
// Contents:Read permission which is not in scope for this GitHub App.
type ReviewComment struct {
	DatabaseID        int64
	Author            string
	Body              string
	CreatedAt         string
	Path              string
	Line              int
	StartLine         int
	OriginalLine      int
	OriginalStartLine int
	DiffHunk          string
	ReplyToID         int64 // 0 if top-level
	IsMinimized       bool
	MinimizedReason   string
	Outdated          bool
	SubjectType       string // LINE or FILE
	Reactions         []format.Reaction
}

// ReviewDetail holds the full review with its inline comments
type ReviewDetail struct {
	DatabaseID  int64
	Author      string
	Body        string
	State       string
	CreatedAt   string
	ViewerLogin string
	Comments    []ReviewComment
	Reactions   []format.Reaction
}

// reactionNode is a single reaction with content and author
type reactionNode struct {
	Content githubv4.String
	User    struct {
		Login githubv4.String
	}
}

// reviewCommentNodeNoDiff - comment query shape without diffHunk
// Note: commit/originalCommit fields require Contents:Read permission and are omitted.
type reviewCommentNodeNoDiff struct {
	DatabaseID int64
	Author     struct {
		Login githubv4.String
	}
	Body              githubv4.String
	CreatedAt         githubv4.DateTime
	Path              githubv4.String
	Line              *githubv4.Int
	StartLine         *githubv4.Int
	OriginalLine      *githubv4.Int
	OriginalStartLine *githubv4.Int
	ReplyTo           *struct {
		DatabaseID int64
	}
	IsMinimized     githubv4.Boolean
	MinimizedReason githubv4.String
	Outdated        githubv4.Boolean
	SubjectType     githubv4.String
	Reactions       struct {
		Nodes []reactionNode
	} `graphql:"reactions(first: 20)"`
}

// reviewCommentNodeWithDiff - comment query shape with diffHunk
// Note: commit/originalCommit fields require Contents:Read permission and are omitted.
type reviewCommentNodeWithDiff struct {
	DatabaseID int64
	Author     struct {
		Login githubv4.String
	}
	Body              githubv4.String
	CreatedAt         githubv4.DateTime
	Path              githubv4.String
	Line              *githubv4.Int
	StartLine         *githubv4.Int
	OriginalLine      *githubv4.Int
	OriginalStartLine *githubv4.Int
	DiffHunk          githubv4.String
	ReplyTo           *struct {
		DatabaseID int64
	}
	IsMinimized     githubv4.Boolean
	MinimizedReason githubv4.String
	Outdated        githubv4.Boolean
	SubjectType     githubv4.String
	Reactions       struct {
		Nodes []reactionNode
	} `graphql:"reactions(first: 20)"`
}

// reviewDetailNodeNoDiff - review node with no-diff comments
type reviewDetailNodeNoDiff struct {
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
	Comments struct {
		Nodes []reviewCommentNodeNoDiff
	} `graphql:"comments(first: 100)"`
}

// reviewDetailNodeWithDiff - review node with diff-included comments
type reviewDetailNodeWithDiff struct {
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
	Comments struct {
		Nodes []reviewCommentNodeWithDiff
	} `graphql:"comments(first: 100)"`
}

// We need to find the review by databaseId, but GraphQL doesn't support filtering by databaseId directly.
// Instead, fetch all reviews and filter client-side.

// allReviewsQueryNoDiff - top-level query without diffHunk
type allReviewsQueryNoDiff struct {
	Viewer struct {
		Login githubv4.String
	}
	Repository struct {
		PullRequest struct {
			Reviews struct {
				Nodes []reviewDetailNodeNoDiff
			} `graphql:"reviews(first: 100)"`
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// allReviewsQueryWithDiff - top-level query including diffHunk
type allReviewsQueryWithDiff struct {
	Viewer struct {
		Login githubv4.String
	}
	Repository struct {
		PullRequest struct {
			Reviews struct {
				Nodes []reviewDetailNodeWithDiff
			} `graphql:"reviews(first: 100)"`
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// Get returns the review detail with all inline comments.
// When showDiff is true, diffHunk is fetched and included in each comment.
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
		for _, n := range query.Repository.PullRequest.Reviews.Nodes {
			if n.DatabaseID == reviewDatabaseID {
				return mapReviewDetailWithDiff(&n, string(query.Viewer.Login)), nil
			}
		}
	} else {
		var query allReviewsQueryNoDiff
		if err := s.gql.Query(context.Background(), &query, variables); err != nil {
			return nil, fmt.Errorf("gql.Query [pr=%d, review=%d]: %w", prNumber, reviewDatabaseID, err)
		}
		for _, n := range query.Repository.PullRequest.Reviews.Nodes {
			if n.DatabaseID == reviewDatabaseID {
				return mapReviewDetailNoDiff(&n, string(query.Viewer.Login)), nil
			}
		}
	}

	return nil, fmt.Errorf("review #%d not found in PR #%d", reviewDatabaseID, prNumber)
}

// mapCommentCore maps common comment fields plus an optional diffHunk into a ReviewComment
func mapCommentCore(
	databaseID int64,
	authorLogin githubv4.String,
	body githubv4.String,
	createdAt githubv4.DateTime,
	path githubv4.String,
	line, startLine, originalLine, originalStartLine *githubv4.Int,
	diffHunk string,
	replyTo *struct{ DatabaseID int64 },
	isMinimized githubv4.Boolean,
	minimizedReason githubv4.String,
	outdated githubv4.Boolean,
	subjectType githubv4.String,
	reactions []reactionNode,
) ReviewComment {
	rc := ReviewComment{
		DatabaseID:      databaseID,
		Author:          string(authorLogin),
		Body:            string(body),
		CreatedAt:       createdAt.UTC().Format("2006-01-02T15:04:05Z"),
		Path:            string(path),
		DiffHunk:        diffHunk,
		IsMinimized:     bool(isMinimized),
		MinimizedReason: string(minimizedReason),
		Outdated:        bool(outdated),
		SubjectType:     string(subjectType),
		Reactions:       mapReactions(reactions),
	}
	if line != nil {
		rc.Line = int(*line)
	}
	if startLine != nil {
		rc.StartLine = int(*startLine)
	}
	if originalLine != nil {
		rc.OriginalLine = int(*originalLine)
	}
	if originalStartLine != nil {
		rc.OriginalStartLine = int(*originalStartLine)
	}
	if replyTo != nil {
		rc.ReplyToID = replyTo.DatabaseID
	}
	return rc
}

func mapCommentNoDiff(c reviewCommentNodeNoDiff) ReviewComment {
	return mapCommentCore(
		c.DatabaseID, c.Author.Login, c.Body, c.CreatedAt, c.Path,
		c.Line, c.StartLine, c.OriginalLine, c.OriginalStartLine,
		"", c.ReplyTo, c.IsMinimized, c.MinimizedReason, c.Outdated, c.SubjectType,
		c.Reactions.Nodes,
	)
}

func mapCommentWithDiff(c reviewCommentNodeWithDiff) ReviewComment {
	return mapCommentCore(
		c.DatabaseID, c.Author.Login, c.Body, c.CreatedAt, c.Path,
		c.Line, c.StartLine, c.OriginalLine, c.OriginalStartLine,
		string(c.DiffHunk), c.ReplyTo, c.IsMinimized, c.MinimizedReason, c.Outdated, c.SubjectType,
		c.Reactions.Nodes,
	)
}

// buildReviewDetail assembles a ReviewDetail from already-mapped comments
func buildReviewDetail(
	databaseID int64,
	authorLogin githubv4.String,
	body githubv4.String,
	state githubv4.String,
	createdAt githubv4.DateTime,
	viewerLogin string,
	reactions []reactionNode,
	comments []ReviewComment,
) *ReviewDetail {
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].CreatedAt < comments[j].CreatedAt
	})
	return &ReviewDetail{
		DatabaseID:  databaseID,
		Author:      string(authorLogin),
		Body:        string(body),
		State:       string(state),
		CreatedAt:   createdAt.UTC().Format("2006-01-02T15:04:05Z"),
		ViewerLogin: viewerLogin,
		Reactions:   mapReactions(reactions),
		Comments:    comments,
	}
}

func mapReviewDetailNoDiff(n *reviewDetailNodeNoDiff, viewerLogin string) *ReviewDetail {
	comments := make([]ReviewComment, len(n.Comments.Nodes))
	for i, c := range n.Comments.Nodes {
		comments[i] = mapCommentNoDiff(c)
	}
	return buildReviewDetail(
		n.DatabaseID, n.Author.Login, n.Body, n.State, n.CreatedAt,
		viewerLogin, n.Reactions.Nodes, comments,
	)
}

func mapReviewDetailWithDiff(n *reviewDetailNodeWithDiff, viewerLogin string) *ReviewDetail {
	comments := make([]ReviewComment, len(n.Comments.Nodes))
	for i, c := range n.Comments.Nodes {
		comments[i] = mapCommentWithDiff(c)
	}
	return buildReviewDetail(
		n.DatabaseID, n.Author.Login, n.Body, n.State, n.CreatedAt,
		viewerLogin, n.Reactions.Nodes, comments,
	)
}

func mapReactions(nodes []reactionNode) []format.Reaction {
	reactions := make([]format.Reaction, len(nodes))
	for i, n := range nodes {
		reactions[i] = format.Reaction{Content: string(n.Content), Author: string(n.User.Login)}
	}
	return reactions
}
