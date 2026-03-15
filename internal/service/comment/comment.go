package comment

import (
	"context"
	"fmt"

	ghrest "github.com/google/go-github/v69/github"
	"github.com/shurcooL/githubv4"

	"github.com/ivanov-gv/gh-contribute/internal/utils/format"
)

// Service provides comment operations — GraphQL for reads, REST for writes
type Service struct {
	gql        *githubv4.Client
	restClient *ghrest.Client
	owner      string
	repo       string
}

// NewService creates a new comment service
func NewService(gql *githubv4.Client, restClient *ghrest.Client, owner, repo string) *Service {
	return &Service{gql: gql, restClient: restClient, owner: owner, repo: repo}
}

// IssueComment holds a top-level PR comment
type IssueComment struct {
	DatabaseID      int64
	Author          string
	Body            string
	CreatedAt       string
	IsMinimized     bool
	MinimizedReason string
	Reactions       []format.Reaction
}

// Review holds a PR review summary
type Review struct {
	DatabaseID      int64
	Author          string
	Body            string
	State           string // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED, PENDING
	CreatedAt       string
	CommentCount    int
	Reactions       []format.Reaction
	IsMinimized     bool
	MinimizedReason string
}

// CommentsResult holds all comments and reviews for a PR
type CommentsResult struct {
	ViewerLogin   string
	IssueComments []IssueComment
	Reviews       []Review
}

// reactionNode is a single reaction with content and author
type reactionNode struct {
	Content githubv4.String
	User    struct {
		Login githubv4.String
	}
}

// issueCommentNode is a single top-level comment node
type issueCommentNode struct {
	DatabaseID      int64
	Author          struct {
		Login githubv4.String
	}
	Body            githubv4.String
	CreatedAt       githubv4.DateTime
	IsMinimized     githubv4.Boolean
	MinimizedReason githubv4.String
	Reactions       struct {
		Nodes []reactionNode
	} `graphql:"reactions(first: 100)"`
}

// reviewNode is a single review node
type reviewNode struct {
	DatabaseID int64
	Author     struct {
		Login githubv4.String
	}
	Body            githubv4.String
	State           githubv4.String
	CreatedAt       githubv4.DateTime
	IsMinimized     githubv4.Boolean
	MinimizedReason githubv4.String
	Comments        struct {
		TotalCount githubv4.Int
	}
	Reactions struct {
		Nodes []reactionNode
	} `graphql:"reactions(first: 100)"`
}

// commentsQuery defines the GraphQL query shape for listing all comments and reviews on a PR
type commentsQuery struct {
	Viewer struct {
		Login githubv4.String
	}
	Repository struct {
		PullRequest struct {
			Comments struct {
				Nodes []issueCommentNode
			} `graphql:"comments(first: 100)"`
			Reviews struct {
				Nodes []reviewNode
			} `graphql:"reviews(first: 100)"`
		} `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// List fetches all issue comments and reviews for a PR
func (s *Service) List(prNumber int) (*CommentsResult, error) {
	var query commentsQuery
	variables := map[string]interface{}{
		"owner":  githubv4.String(s.owner),
		"repo":   githubv4.String(s.repo),
		"number": githubv4.Int(prNumber),
	}
	if err := s.gql.Query(context.Background(), &query, variables); err != nil {
		return nil, fmt.Errorf("gql.Query [pr=%d]: %w", prNumber, err)
	}

	pr := query.Repository.PullRequest

	var issueComments []IssueComment
	for _, n := range pr.Comments.Nodes {
		issueComments = append(issueComments, IssueComment{
			DatabaseID:      n.DatabaseID,
			Author:          string(n.Author.Login),
			Body:            string(n.Body),
			CreatedAt:       n.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			IsMinimized:     bool(n.IsMinimized),
			MinimizedReason: string(n.MinimizedReason),
			Reactions:       mapReactions(n.Reactions.Nodes),
		})
	}

	var reviews []Review
	for _, n := range pr.Reviews.Nodes {
		reviews = append(reviews, Review{
			DatabaseID:      n.DatabaseID,
			Author:          string(n.Author.Login),
			Body:            string(n.Body),
			State:           string(n.State),
			CreatedAt:       n.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			CommentCount:    int(n.Comments.TotalCount),
			Reactions:       mapReactions(n.Reactions.Nodes),
			IsMinimized:     bool(n.IsMinimized),
			MinimizedReason: string(n.MinimizedReason),
		})
	}

	return &CommentsResult{
		ViewerLogin:   string(query.Viewer.Login),
		IssueComments: issueComments,
		Reviews:       reviews,
	}, nil
}

// Post creates a new top-level comment on a PR via REST API
func (s *Service) Post(prNumber int, body string) (*IssueComment, error) {
	comment, _, err := s.restClient.Issues.CreateComment(
		context.Background(), s.owner, s.repo, prNumber,
		&ghrest.IssueComment{Body: ghrest.Ptr(body)},
	)
	if err != nil {
		return nil, fmt.Errorf("Issues.CreateComment [pr=%d]: %w", prNumber, err)
	}
	return &IssueComment{
		DatabaseID: comment.GetID(),
		Author:     comment.GetUser().GetLogin(),
		Body:       comment.GetBody(),
		CreatedAt:  comment.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
	}, nil
}

// FilterByID returns a CommentsResult containing only the comment or review with the given database ID
func (r *CommentsResult) FilterByID(id int64) *CommentsResult {
	for _, c := range r.IssueComments {
		if c.DatabaseID == id {
			return &CommentsResult{
				ViewerLogin:   r.ViewerLogin,
				IssueComments: []IssueComment{c},
			}
		}
	}
	for _, rev := range r.Reviews {
		if rev.DatabaseID == id {
			return &CommentsResult{
				ViewerLogin: r.ViewerLogin,
				Reviews:     []Review{rev},
			}
		}
	}
	return nil
}

func mapReactions(nodes []reactionNode) []format.Reaction {
	reactions := make([]format.Reaction, len(nodes))
	for i, n := range nodes {
		reactions[i] = format.Reaction{Content: string(n.Content), Author: string(n.User.Login)}
	}
	return reactions
}
