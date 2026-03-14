package comment

import (
	"context"
	"fmt"

	ghrest "github.com/google/go-github/v69/github"

	"github.com/ivanov-gv/gh-contribute/internal/format"
	gh "github.com/ivanov-gv/gh-contribute/internal/github"
)

// Service provides comment operations — GraphQL for reads, REST for writes
type Service struct {
	gql        *gh.GraphQLClient
	restClient *ghrest.Client
	owner      string
	repo       string
}

// NewService creates a new comment service
func NewService(gql *gh.GraphQLClient, restClient *ghrest.Client, owner, repo string) *Service {
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

// commentsQuery is the GraphQL response shape
type commentsQuery struct {
	Viewer struct {
		Login string `json:"login"`
	} `json:"viewer"`
	Repository struct {
		PullRequest struct {
			Comments struct {
				Nodes []issueCommentNode `json:"nodes"`
			} `json:"comments"`
			Reviews struct {
				Nodes []reviewNode `json:"nodes"`
			} `json:"reviews"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

type issueCommentNode struct {
	DatabaseID      int64  `json:"databaseId"`
	Author          author `json:"author"`
	Body            string `json:"body"`
	CreatedAt       string `json:"createdAt"`
	IsMinimized     bool   `json:"isMinimized"`
	MinimizedReason string `json:"minimizedReason"`
	Reactions       struct {
		Nodes []reactionNode `json:"nodes"`
	} `json:"reactions"`
}

type reviewNode struct {
	DatabaseID      int64  `json:"databaseId"`
	Author          author `json:"author"`
	Body            string `json:"body"`
	State           string `json:"state"`
	CreatedAt       string `json:"createdAt"`
	IsMinimized     bool   `json:"isMinimized"`
	MinimizedReason string `json:"minimizedReason"`
	Comments        struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	Reactions struct {
		Nodes []reactionNode `json:"nodes"`
	} `json:"reactions"`
}

type author struct {
	Login string `json:"login"`
}

type reactionNode struct {
	Content string `json:"content"`
	User    author `json:"user"`
}

const commentsQueryString = `
query($owner: String!, $repo: String!, $number: Int!) {
  viewer { login }
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      comments(first: 100) {
        nodes {
          databaseId
          author { login }
          body createdAt
          isMinimized minimizedReason
          reactions(first: 100) {
            nodes { content user { login } }
          }
        }
      }
      reviews(first: 100) {
        nodes {
          databaseId
          author { login }
          body state createdAt
          isMinimized minimizedReason
          comments { totalCount }
          reactions(first: 100) {
            nodes { content user { login } }
          }
        }
      }
    }
  }
}`

// List fetches all issue comments and reviews for a PR
func (s *Service) List(prNumber int) (*CommentsResult, error) {
	var result commentsQuery
	err := s.gql.Query(commentsQueryString, map[string]interface{}{
		"owner":  s.owner,
		"repo":   s.repo,
		"number": prNumber,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("GraphQL query [pr=%d]: %w", prNumber, err)
	}

	pr := result.Repository.PullRequest

	var issueComments []IssueComment
	for _, n := range pr.Comments.Nodes {
		issueComments = append(issueComments, IssueComment{
			DatabaseID:      n.DatabaseID,
			Author:          n.Author.Login,
			Body:            n.Body,
			CreatedAt:       n.CreatedAt,
			IsMinimized:     n.IsMinimized,
			MinimizedReason: n.MinimizedReason,
			Reactions:       mapReactions(n.Reactions.Nodes),
		})
	}

	var reviews []Review
	for _, n := range pr.Reviews.Nodes {
		reviews = append(reviews, Review{
			DatabaseID:      n.DatabaseID,
			Author:          n.Author.Login,
			Body:            n.Body,
			State:           n.State,
			CreatedAt:       n.CreatedAt,
			CommentCount:    n.Comments.TotalCount,
			Reactions:       mapReactions(n.Reactions.Nodes),
			IsMinimized:     n.IsMinimized,
			MinimizedReason: n.MinimizedReason,
		})
	}

	return &CommentsResult{
		ViewerLogin:   result.Viewer.Login,
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
		reactions[i] = format.Reaction{Content: n.Content, Author: n.User.Login}
	}
	return reactions
}
