package comment

import (
	"fmt"

	ghrest "github.com/google/go-github/v69/github"

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

// Reaction holds a single reaction with its author
type Reaction struct {
	Content string // GraphQL enum: THUMBS_UP, ROCKET, EYES, etc.
	Author  string
}

// IssueComment holds a top-level PR comment
type IssueComment struct {
	DatabaseID      int64
	Author          string
	Body            string
	CreatedAt       string
	IsMinimized     bool
	MinimizedReason string
	Reactions       []Reaction
}

// Review holds a PR review summary
type Review struct {
	DatabaseID   int64
	Author       string
	Body         string
	State        string // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED, PENDING
	CreatedAt    string
	CommentCount int
	Reactions    []Reaction
	AllResolved  bool // true if all review threads are resolved
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
			ReviewThreads struct {
				Nodes []reviewThreadNode `json:"nodes"`
			} `json:"reviewThreads"`
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
	DatabaseID int64  `json:"databaseId"`
	Author     author `json:"author"`
	Body       string `json:"body"`
	State      string `json:"state"`
	CreatedAt  string `json:"createdAt"`
	Comments   struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	Reactions struct {
		Nodes []reactionNode `json:"nodes"`
	} `json:"reactions"`
}

type reviewThreadNode struct {
	IsResolved bool `json:"isResolved"`
	Comments   struct {
		Nodes []struct {
			PullRequestReview *struct {
				DatabaseID int64 `json:"databaseId"`
			} `json:"pullRequestReview"`
		} `json:"nodes"`
	} `json:"comments"`
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
          comments { totalCount }
          reactions(first: 100) {
            nodes { content user { login } }
          }
        }
      }
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 1) {
            nodes {
              pullRequestReview { databaseId }
            }
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

	// build map: reviewDatabaseId → list of thread isResolved values
	reviewThreadResolved := make(map[int64][]bool)
	for _, thread := range pr.ReviewThreads.Nodes {
		if len(thread.Comments.Nodes) > 0 && thread.Comments.Nodes[0].PullRequestReview != nil {
			reviewID := thread.Comments.Nodes[0].PullRequestReview.DatabaseID
			reviewThreadResolved[reviewID] = append(reviewThreadResolved[reviewID], thread.IsResolved)
		}
	}

	// map issue comments
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

	// map reviews
	var reviews []Review
	for _, n := range pr.Reviews.Nodes {
		// check if all threads for this review are resolved
		allResolved := false
		if threads, ok := reviewThreadResolved[n.DatabaseID]; ok && len(threads) > 0 {
			allResolved = true
			for _, resolved := range threads {
				if !resolved {
					allResolved = false
					break
				}
			}
		}

		reviews = append(reviews, Review{
			DatabaseID:   n.DatabaseID,
			Author:       n.Author.Login,
			Body:         n.Body,
			State:        n.State,
			CreatedAt:    n.CreatedAt,
			CommentCount: n.Comments.TotalCount,
			Reactions:    mapReactions(n.Reactions.Nodes),
			AllResolved:  allResolved,
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
		gh.Context(), s.owner, s.repo, prNumber,
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

func mapReactions(nodes []reactionNode) []Reaction {
	var reactions []Reaction
	for _, n := range nodes {
		reactions = append(reactions, Reaction{
			Content: n.Content,
			Author:  n.User.Login,
		})
	}
	return reactions
}
