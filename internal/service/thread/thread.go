package thread

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"

	"github.com/ivanov-gv/gh-contribute/internal/utils/format"
)

// Service provides thread lookup operations via GraphQL
type Service struct {
	gql   *githubv4.Client
	owner string
	repo  string
}

// NewService creates a new thread service
func NewService(gql *githubv4.Client, owner, repo string) *Service {
	return &Service{gql: gql, owner: owner, repo: repo}
}

// ThreadComment holds a single comment in a thread across all reviews
type ThreadComment struct {
	DatabaseID       int64
	Author           string
	Body             string
	CreatedAt        string
	ReviewDatabaseID int64 // which review this comment belongs to
	ReplyToID        int64 // 0 if thread root
	IsMinimized      bool
	MinimizedReason  string
	Reactions        []format.Reaction
}

// Thread holds all comments in a thread and location info
type Thread struct {
	ThreadID          int64 // databaseId of the first comment
	IsOutdated        bool
	Path              string
	Line              int
	StartLine         int
	OriginalLine      int
	OriginalStartLine int
	ViewerLogin       string
	Comments          []ThreadComment
}

// reactionNode is a single reaction with content and author
type reactionNode struct {
	Content githubv4.String
	User    struct {
		Login githubv4.String
	}
}

// threadCommentNode represents a comment within a review thread
type threadCommentNode struct {
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

// reviewThreadNode represents a single review thread with its comments
type reviewThreadNode struct {
	IsOutdated        githubv4.Boolean
	Path              githubv4.String
	Line              *githubv4.Int
	StartLine         *githubv4.Int
	OriginalLine      *githubv4.Int
	OriginalStartLine *githubv4.Int
	Comments          struct {
		Nodes []threadCommentNode
	} `graphql:"comments(first: 50)"`
}

// threadsQuery fetches all review threads for a PR
type threadsQuery struct {
	Viewer struct {
		Login githubv4.String
	}
	Repository struct {
		PullRequest struct {
			ReviewThreads struct {
				Nodes []reviewThreadNode
			} `graphql:"reviewThreads(first: 100)"`
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// Get returns the full thread identified by threadID (the databaseId of the first comment).
func (s *Service) Get(prNumber int, threadID int64) (*Thread, error) {
	variables := map[string]interface{}{
		"owner":  githubv4.String(s.owner),
		"repo":   githubv4.String(s.repo),
		"number": githubv4.Int(prNumber),
	}

	var query threadsQuery
	if err := s.gql.Query(context.Background(), &query, variables); err != nil {
		return nil, fmt.Errorf("gql.Query [pr=%d, thread=%d]: %w", prNumber, threadID, err)
	}

	viewerLogin := string(query.Viewer.Login)
	for _, n := range query.Repository.PullRequest.ReviewThreads.Nodes {
		if len(n.Comments.Nodes) == 0 || n.Comments.Nodes[0].DatabaseID != threadID {
			continue
		}
		return buildThread(n, viewerLogin, threadID), nil
	}

	return nil, fmt.Errorf("thread #%d not found in PR #%d", threadID, prNumber)
}

func buildThread(n reviewThreadNode, viewerLogin string, threadID int64) *Thread {
	t := &Thread{
		ThreadID:    threadID,
		IsOutdated:  bool(n.IsOutdated),
		Path:        string(n.Path),
		ViewerLogin: viewerLogin,
	}
	if n.Line != nil {
		t.Line = int(*n.Line)
	}
	if n.StartLine != nil {
		t.StartLine = int(*n.StartLine)
	}
	if n.OriginalLine != nil {
		t.OriginalLine = int(*n.OriginalLine)
	}
	if n.OriginalStartLine != nil {
		t.OriginalStartLine = int(*n.OriginalStartLine)
	}

	for _, c := range n.Comments.Nodes {
		tc := ThreadComment{
			DatabaseID:      c.DatabaseID,
			Author:          string(c.Author.Login),
			Body:            string(c.Body),
			CreatedAt:       c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			IsMinimized:     bool(c.IsMinimized),
			MinimizedReason: string(c.MinimizedReason),
			Reactions:       mapReactions(c.Reactions.Nodes),
		}
		if c.ReplyTo != nil {
			tc.ReplyToID = c.ReplyTo.DatabaseID
		}
		if c.PullRequestReview != nil {
			tc.ReviewDatabaseID = c.PullRequestReview.DatabaseID
		}
		t.Comments = append(t.Comments, tc)
	}
	return t
}

func mapReactions(nodes []reactionNode) []format.Reaction {
	reactions := make([]format.Reaction, len(nodes))
	for i, n := range nodes {
		reactions[i] = format.Reaction{Content: string(n.Content), Author: string(n.User.Login)}
	}
	return reactions
}
