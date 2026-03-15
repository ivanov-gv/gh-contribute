package pr

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
)

// Service provides PR operations via GraphQL
type Service struct {
	gql   *githubv4.Client
	owner string
	repo  string
}

// NewService creates a new PR service
func NewService(gql *githubv4.Client, owner, repo string) *Service {
	return &Service{gql: gql, owner: owner, repo: repo}
}

// Info holds rich PR details from GraphQL
type Info struct {
	Number       int
	Title        string
	State        string
	IsDraft      bool
	Mergeable    string
	Body         string
	URL          string
	Head         string
	Base         string
	Author       string
	CommitCount  int
	CommentCount int
	Reviewers    []string
	Assignees    []string
	Labels       []string
	Projects     []string
	Milestone    string
	Issues       []LinkedIssue
}

// LinkedIssue is an issue referenced by the PR
type LinkedIssue struct {
	Number int
	Title  string
}

// prMilestone is a nullable milestone node
type prMilestone struct {
	Title githubv4.String
}

// prReviewerNode handles both User and Team reviewer types via inline fragments
type prReviewerNode struct {
	User struct {
		Login githubv4.String
	} `graphql:"... on User"`
	Team struct {
		Name githubv4.String
	} `graphql:"... on Team"`
}

// prNode is the pull request shape returned by the query
type prNode struct {
	Number      githubv4.Int
	Title       githubv4.String
	State       githubv4.String
	IsDraft     githubv4.Boolean
	Mergeable   githubv4.String
	Body        githubv4.String
	URL         githubv4.URI
	HeadRefName githubv4.String
	BaseRefName githubv4.String
	Author      struct {
		Login githubv4.String
	}
	Commits struct {
		TotalCount githubv4.Int
	}
	Comments struct {
		TotalCount githubv4.Int
	}
	Reviews struct {
		TotalCount githubv4.Int
	}
	Assignees struct {
		Nodes []struct {
			Login githubv4.String
		}
	} `graphql:"assignees(first: 20)"`
	Labels struct {
		Nodes []struct {
			Name githubv4.String
		}
	} `graphql:"labels(first: 20)"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer prReviewerNode
		}
	} `graphql:"reviewRequests(first: 20)"`
	Milestone  *prMilestone
	ProjectsV2 struct {
		Nodes []struct {
			Title githubv4.String
		}
	} `graphql:"projectsV2(first: 10)"`
	ClosingIssuesReferences struct {
		Nodes []struct {
			Number githubv4.Int
			Title  githubv4.String
		}
	} `graphql:"closingIssuesReferences(first: 20)"`
}

// prQuery defines the GraphQL query shape for a PR by number
type prQuery struct {
	Repository struct {
		PullRequest prNode `graphql:"pullRequest(number: $number)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// Get returns rich PR info by number
func (s *Service) Get(number int) (*Info, error) {
	var query prQuery
	variables := map[string]interface{}{
		"owner":  githubv4.String(s.owner),
		"repo":   githubv4.String(s.repo),
		"number": githubv4.Int(number),
	}
	if err := s.gql.Query(context.Background(), &query, variables); err != nil {
		return nil, fmt.Errorf("gql.Query [number=%d]: %w", number, err)
	}
	return mapPR(&query.Repository.PullRequest), nil
}

// findByBranchQuery defines the GraphQL query shape for finding a PR by branch
type findByBranchQuery struct {
	Repository struct {
		PullRequests struct {
			Nodes []struct {
				Number githubv4.Int
			}
		} `graphql:"pullRequests(first: 1, headRefName: $head, states: OPEN)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}

// FindByBranch finds an open PR number for the given head branch
func (s *Service) FindByBranch(branch string) (int, error) {
	var query findByBranchQuery
	variables := map[string]interface{}{
		"owner": githubv4.String(s.owner),
		"repo":  githubv4.String(s.repo),
		"head":  githubv4.String(branch),
	}
	if err := s.gql.Query(context.Background(), &query, variables); err != nil {
		return 0, fmt.Errorf("gql.Query [branch='%s']: %w", branch, err)
	}
	nodes := query.Repository.PullRequests.Nodes
	if len(nodes) == 0 {
		return 0, fmt.Errorf("no open PR found for branch '%s'", branch)
	}
	return int(nodes[0].Number), nil
}

// mapPR converts the GraphQL response to our Info type
func mapPR(n *prNode) *Info {
	info := &Info{
		Number:       int(n.Number),
		Title:        string(n.Title),
		State:        strings.ToLower(string(n.State)),
		IsDraft:      bool(n.IsDraft),
		Mergeable:    string(n.Mergeable),
		Body:         string(n.Body),
		URL:          n.URL.String(),
		Head:         string(n.HeadRefName),
		Base:         string(n.BaseRefName),
		Author:       string(n.Author.Login),
		CommitCount:  int(n.Commits.TotalCount),
		CommentCount: int(n.Comments.TotalCount) + int(n.Reviews.TotalCount),
	}

	// reviewers
	for _, rr := range n.ReviewRequests.Nodes {
		if login := string(rr.RequestedReviewer.User.Login); login != "" {
			info.Reviewers = append(info.Reviewers, "@"+login)
		} else if name := string(rr.RequestedReviewer.Team.Name); name != "" {
			info.Reviewers = append(info.Reviewers, name)
		}
	}

	// assignees
	for _, a := range n.Assignees.Nodes {
		info.Assignees = append(info.Assignees, "@"+string(a.Login))
	}

	// labels
	for _, l := range n.Labels.Nodes {
		info.Labels = append(info.Labels, string(l.Name))
	}

	// projects
	for _, p := range n.ProjectsV2.Nodes {
		info.Projects = append(info.Projects, string(p.Title))
	}

	// milestone
	if n.Milestone != nil {
		info.Milestone = string(n.Milestone.Title)
	}

	// linked issues
	for _, i := range n.ClosingIssuesReferences.Nodes {
		info.Issues = append(info.Issues, LinkedIssue{Number: int(i.Number), Title: string(i.Title)})
	}

	return info
}
