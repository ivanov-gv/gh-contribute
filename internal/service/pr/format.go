package pr

import (
	"fmt"
	"strings"
)

// Format renders PR info as human-readable markdown
func (info *Info) Format() string {
	var b strings.Builder

	// header: # title #number
	b.WriteString(fmt.Sprintf("# %s #%d\n", info.Title, info.Number))

	// meta line: state, author, commits, branches, merge conflict
	state := info.State
	if info.IsDraft {
		state = "draft"
	}
	commitWord := "commits"
	if info.CommitCount == 1 {
		commitWord = "commit"
	}
	mergeStatus := "no merge conflict"
	if info.Mergeable == "CONFLICTING" {
		mergeStatus = "merge conflict"
	} else if info.Mergeable == "UNKNOWN" {
		mergeStatus = "merge status unknown"
	}
	b.WriteString(fmt.Sprintf("%s, by @%s, %d %s `%s` -> `%s`, %s\n",
		state, info.Author, info.CommitCount, commitWord, info.Head, info.Base, mergeStatus))

	// url
	b.WriteString(info.URL + "\n")
	b.WriteString("\n")

	// metadata fields
	b.WriteString(fmt.Sprintf("Reviewers: %s  \n", strings.Join(info.Reviewers, ", ")))
	b.WriteString(fmt.Sprintf("Assignees: %s  \n", strings.Join(info.Assignees, ", ")))
	b.WriteString(fmt.Sprintf("Labels: %s  \n", strings.Join(info.Labels, ", ")))
	b.WriteString(fmt.Sprintf("Projects: %s  \n", strings.Join(info.Projects, ", ")))
	b.WriteString(fmt.Sprintf("Milestone: %s  \n", info.Milestone))

	// linked issues
	var issueStrs []string
	for _, i := range info.Issues {
		issueStrs = append(issueStrs, fmt.Sprintf("#%d %s", i.Number, i.Title))
	}
	b.WriteString(fmt.Sprintf("Issues: %s  \n", strings.Join(issueStrs, ", ")))

	// description
	b.WriteString("\n---\n\n")
	body := strings.TrimSpace(info.Body)
	if body == "" {
		body = "No description provided."
	}
	b.WriteString(body + "\n")
	b.WriteString("\n---\n")

	return b.String()
}
