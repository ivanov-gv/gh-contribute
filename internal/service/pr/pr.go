package pr

import (
	"fmt"
	"strings"

	gh "github.com/ivanov-gv/gh-contribute/internal/github"
)

// Service provides PR operations via GraphQL
type Service struct {
	gql   *gh.GraphQLClient
	owner string
	repo  string
}

// NewService creates a new PR service
func NewService(gql *gh.GraphQLClient, owner, repo string) *Service {
	return &Service{gql: gql, owner: owner, repo: repo}
}

// Info holds rich PR details from GraphQL
type Info struct {
	Number      int
	Title       string
	State       string
	IsDraft     bool
	Mergeable   string
	Body        string
	URL         string
	Head        string
	Base        string
	Author      string
	CommitCount int
	Reviewers   []string
	Assignees   []string
	Labels      []string
	Projects    []string
	Milestone   string
	Issues      []LinkedIssue
}

// LinkedIssue is an issue referenced by the PR
type LinkedIssue struct {
	Number int
	Title  string
}

// prQuery is the GraphQL response shape for a PR
type prQuery struct {
	Repository struct {
		PullRequest prNode `json:"pullRequest"`
	} `json:"repository"`
}

type prNode struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	IsDraft     bool   `json:"isDraft"`
	Mergeable   string `json:"mergeable"`
	Body        string `json:"body"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
	Commits struct {
		TotalCount int `json:"totalCount"`
	} `json:"commits"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer reviewerUnion `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	ProjectsV2 struct {
		Nodes []struct {
			Title string `json:"title"`
		} `json:"nodes"`
	} `json:"projectsV2"`
	ClosingIssuesReferences struct {
		Nodes []struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
		} `json:"nodes"`
	} `json:"closingIssuesReferences"`
}

// reviewerUnion handles both User and Team reviewer types
type reviewerUnion struct {
	Login string `json:"login"` // User
	Name  string `json:"name"`  // Team
}

const prQueryString = `
query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      number title state isDraft mergeable body url
      headRefName baseRefName
      author { login }
      commits { totalCount }
      assignees(first: 20) { nodes { login } }
      labels(first: 20) { nodes { name } }
      reviewRequests(first: 20) {
        nodes { requestedReviewer { ... on User { login } ... on Team { name } } }
      }
      milestone { title }
      projectsV2(first: 10) { nodes { title } }
      closingIssuesReferences(first: 20) { nodes { number title } }
    }
  }
}`

// Get returns rich PR info by number
func (s *Service) Get(number int) (*Info, error) {
	var result prQuery
	err := s.gql.Query(prQueryString, map[string]interface{}{
		"owner":  s.owner,
		"repo":   s.repo,
		"number": number,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("GraphQL query [number=%d]: %w", number, err)
	}
	return mapPR(&result.Repository.PullRequest), nil
}

// findByBranchQuery is the GraphQL response shape for finding a PR by branch
type findByBranchQuery struct {
	Repository struct {
		PullRequests struct {
			Nodes []struct {
				Number int `json:"number"`
			} `json:"nodes"`
		} `json:"pullRequests"`
	} `json:"repository"`
}

const findByBranchQueryString = `
query($owner: String!, $repo: String!, $head: String!) {
  repository(owner: $owner, name: $repo) {
    pullRequests(first: 1, headRefName: $head, states: OPEN) {
      nodes { number }
    }
  }
}`

// FindByBranch finds an open PR number for the given head branch
func (s *Service) FindByBranch(branch string) (int, error) {
	var result findByBranchQuery
	err := s.gql.Query(findByBranchQueryString, map[string]interface{}{
		"owner": s.owner,
		"repo":  s.repo,
		"head":  branch,
	}, &result)
	if err != nil {
		return 0, fmt.Errorf("GraphQL query [branch='%s']: %w", branch, err)
	}
	nodes := result.Repository.PullRequests.Nodes
	if len(nodes) == 0 {
		return 0, fmt.Errorf("no open PR found for branch '%s'", branch)
	}
	return nodes[0].Number, nil
}

// mapPR converts the GraphQL response to our Info type
func mapPR(n *prNode) *Info {
	info := &Info{
		Number:      n.Number,
		Title:       n.Title,
		State:       strings.ToLower(n.State),
		IsDraft:     n.IsDraft,
		Mergeable:   n.Mergeable,
		Body:        n.Body,
		URL:         n.URL,
		Head:        n.HeadRefName,
		Base:        n.BaseRefName,
		Author:      n.Author.Login,
		CommitCount: n.Commits.TotalCount,
	}

	// reviewers
	for _, rr := range n.ReviewRequests.Nodes {
		if rr.RequestedReviewer.Login != "" {
			info.Reviewers = append(info.Reviewers, "@"+rr.RequestedReviewer.Login)
		} else if rr.RequestedReviewer.Name != "" {
			info.Reviewers = append(info.Reviewers, rr.RequestedReviewer.Name)
		}
	}

	// assignees
	for _, a := range n.Assignees.Nodes {
		info.Assignees = append(info.Assignees, "@"+a.Login)
	}

	// labels
	for _, l := range n.Labels.Nodes {
		info.Labels = append(info.Labels, l.Name)
	}

	// projects
	for _, p := range n.ProjectsV2.Nodes {
		info.Projects = append(info.Projects, p.Title)
	}

	// milestone
	if n.Milestone != nil {
		info.Milestone = n.Milestone.Title
	}

	// linked issues
	for _, i := range n.ClosingIssuesReferences.Nodes {
		info.Issues = append(info.Issues, LinkedIssue{Number: i.Number, Title: i.Title})
	}

	return info
}
