package review

import (
	"fmt"
	"sort"

	"github.com/ivanov-gv/gh-contribute/internal/format"
	gh "github.com/ivanov-gv/gh-contribute/internal/github"
)

// Service provides review detail operations via GraphQL
type Service struct {
	gql   *gh.GraphQLClient
	owner string
	repo  string
}

// NewService creates a new review service
func NewService(gql *gh.GraphQLClient, owner, repo string) *Service {
	return &Service{gql: gql, owner: owner, repo: repo}
}

// ReviewComment holds a single inline review comment
type ReviewComment struct {
	DatabaseID      int64
	Author          string
	Body            string
	CreatedAt       string
	Path            string
	Line            int
	StartLine       int
	DiffHunk        string
	ReplyToID       int64 // 0 if top-level
	IsMinimized     bool
	MinimizedReason string
	Outdated        bool
	SubjectType     string // LINE or FILE
	Reactions       []format.Reaction
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

// reviewDetailQuery is the GraphQL response shape
type reviewDetailQuery struct {
	Viewer struct {
		Login string `json:"login"`
	} `json:"viewer"`
	Repository struct {
		PullRequest struct {
			Reviews struct {
				Nodes []reviewDetailNode `json:"nodes"`
			} `json:"reviews"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

type reviewDetailNode struct {
	DatabaseID int64 `json:"databaseId"`
	Author     struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string `json:"body"`
	State     string `json:"state"`
	CreatedAt string `json:"createdAt"`
	Comments  struct {
		Nodes []reviewCommentNode `json:"nodes"`
	} `json:"comments"`
	Reactions struct {
		Nodes []reactionNode `json:"nodes"`
	} `json:"reactions"`
}

type reviewCommentNode struct {
	DatabaseID int64 `json:"databaseId"`
	Author     struct {
		Login string `json:"login"`
	} `json:"author"`
	Body            string `json:"body"`
	CreatedAt       string `json:"createdAt"`
	Path            string `json:"path"`
	Line            *int   `json:"line"`
	StartLine       *int   `json:"startLine"`
	DiffHunk        string `json:"diffHunk"`
	ReplyTo         *struct {
		DatabaseID int64 `json:"databaseId"`
	} `json:"replyTo"`
	IsMinimized     bool   `json:"isMinimized"`
	MinimizedReason string `json:"minimizedReason"`
	Outdated        bool   `json:"outdated"`
	SubjectType     string `json:"subjectType"`
	Reactions       struct {
		Nodes []reactionNode `json:"nodes"`
	} `json:"reactions"`
}

type reactionNode struct {
	Content string `json:"content"`
	User    struct {
		Login string `json:"login"`
	} `json:"user"`
}

// We need to find the review by databaseId, but GraphQL doesn't support filtering by databaseId directly.
// Instead, fetch all reviews and filter client-side.
const allReviewsQueryString = `
query($owner: String!, $repo: String!, $number: Int!) {
  viewer { login }
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      reviews(first: 100) {
        nodes {
          databaseId
          author { login }
          body state createdAt
          reactions(first: 20) {
            nodes { content user { login } }
          }
          comments(first: 100) {
            nodes {
              databaseId
              author { login }
              body createdAt path line startLine diffHunk
              replyTo { databaseId }
              isMinimized minimizedReason outdated subjectType
              reactions(first: 20) {
                nodes { content user { login } }
              }
            }
          }
        }
      }
    }
  }
}`

// Get returns the review detail with all inline comments
func (s *Service) Get(prNumber int, reviewDatabaseID int64) (*ReviewDetail, error) {
	var result reviewDetailQuery
	err := s.gql.Query(allReviewsQueryString, map[string]interface{}{
		"owner":  s.owner,
		"repo":   s.repo,
		"number": prNumber,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("GraphQL query [pr=%d, review=%d]: %w", prNumber, reviewDatabaseID, err)
	}

	for _, n := range result.Repository.PullRequest.Reviews.Nodes {
		if n.DatabaseID == reviewDatabaseID {
			return mapReviewDetail(&n, result.Viewer.Login), nil
		}
	}

	return nil, fmt.Errorf("review #%d not found in PR #%d", reviewDatabaseID, prNumber)
}

func mapReviewDetail(n *reviewDetailNode, viewerLogin string) *ReviewDetail {
	detail := &ReviewDetail{
		DatabaseID:  n.DatabaseID,
		Author:      n.Author.Login,
		Body:        n.Body,
		State:       n.State,
		CreatedAt:   n.CreatedAt,
		ViewerLogin: viewerLogin,
		Reactions:   mapReactions(n.Reactions.Nodes),
	}

	for _, c := range n.Comments.Nodes {
		rc := ReviewComment{
			DatabaseID:      c.DatabaseID,
			Author:          c.Author.Login,
			Body:            c.Body,
			CreatedAt:       c.CreatedAt,
			Path:            c.Path,
			DiffHunk:        c.DiffHunk,
			IsMinimized:     c.IsMinimized,
			MinimizedReason: c.MinimizedReason,
			Outdated:        c.Outdated,
			SubjectType:     c.SubjectType,
			Reactions:       mapReactions(c.Reactions.Nodes),
		}
		if c.Line != nil {
			rc.Line = *c.Line
		}
		if c.StartLine != nil {
			rc.StartLine = *c.StartLine
		}
		if c.ReplyTo != nil {
			rc.ReplyToID = c.ReplyTo.DatabaseID
		}
		detail.Comments = append(detail.Comments, rc)
	}

	sort.Slice(detail.Comments, func(i, j int) bool {
		return detail.Comments[i].CreatedAt < detail.Comments[j].CreatedAt
	})

	return detail
}

func mapReactions(nodes []reactionNode) []format.Reaction {
	reactions := make([]format.Reaction, len(nodes))
	for i, n := range nodes {
		reactions[i] = format.Reaction{Content: n.Content, Author: n.User.Login}
	}
	return reactions
}
