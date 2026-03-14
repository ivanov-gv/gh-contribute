package comment

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ivanov-gv/gh-contribute/internal/format"
)

// timelineEntry is a union type for sorting issue comments and reviews together
type timelineEntry struct {
	createdAt    string
	issueComment *IssueComment
	review       *Review
}

// Format renders the full comments result as human-readable markdown
func (r *CommentsResult) Format() string {
	// merge issue comments and reviews into a single timeline
	var entries []timelineEntry
	for i := range r.IssueComments {
		entries = append(entries, timelineEntry{
			createdAt:    r.IssueComments[i].CreatedAt,
			issueComment: &r.IssueComments[i],
		})
	}
	for i := range r.Reviews {
		entries = append(entries, timelineEntry{
			createdAt: r.Reviews[i].CreatedAt,
			review:    &r.Reviews[i],
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].createdAt < entries[j].createdAt
	})

	var b strings.Builder
	for i, e := range entries {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		if e.issueComment != nil {
			b.WriteString(formatIssueComment(e.issueComment, r.ViewerLogin))
		} else {
			b.WriteString(formatReview(e.review, r.ViewerLogin))
		}
	}

	return b.String()
}

func formatIssueComment(c *IssueComment, viewerLogin string) string {
	var b strings.Builder

	authorDisplay := format.Author(c.Author, viewerLogin)
	if c.IsMinimized {
		reason := format.EnumLabel(c.MinimizedReason)
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("issue #%d by %s | hidden: %s\n", c.DatabaseID, authorDisplay, reason))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("issue #%d by %s  \n", c.DatabaseID, authorDisplay))
	b.WriteString(fmt.Sprintf("_%s_  \n", format.Date(c.CreatedAt)))
	b.WriteString("\n")
	b.WriteString(c.Body + "\n")
	b.WriteString("\n")
	b.WriteString(format.Reactions(c.Reactions, viewerLogin))

	return b.String()
}

func formatReview(r *Review, viewerLogin string) string {
	var b strings.Builder

	authorDisplay := format.Author(r.Author, viewerLogin)

	if r.IsMinimized {
		reason := format.EnumLabel(r.MinimizedReason)
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("review #%d by %s | hidden: %s\n", r.DatabaseID, authorDisplay, reason))
		return b.String()
	}
	if r.State == "DISMISSED" {
		b.WriteString(fmt.Sprintf("review #%d by %s | hidden: Dismissed\n", r.DatabaseID, authorDisplay))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("review #%d by %s  \n", r.DatabaseID, authorDisplay))
	b.WriteString(fmt.Sprintf("_%s_  \n", format.Date(r.CreatedAt)))
	b.WriteString("\n")

	if r.Body != "" {
		b.WriteString(r.Body + "\n")
		b.WriteString("\n")
	}

	if r.CommentCount > 0 {
		b.WriteString(fmt.Sprintf("comments: %d  \n", r.CommentCount))
	}

	b.WriteString(format.Reactions(r.Reactions, viewerLogin))

	return b.String()
}
