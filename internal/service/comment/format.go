package comment

import (
	"fmt"
	"sort"
	"strings"
)

// reactionEmoji maps GraphQL reaction content enums to emoji
var reactionEmoji = map[string]string{
	"THUMBS_UP":   "👍",
	"THUMBS_DOWN": "👎",
	"LAUGH":       "😄",
	"HOORAY":      "🎉",
	"CONFUSED":    "😕",
	"HEART":       "❤️",
	"ROCKET":      "🚀",
	"EYES":        "👀",
}

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

	// sort by date
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].createdAt < entries[j].createdAt
	})

	var b strings.Builder
	for i, e := range entries {
		if e.issueComment != nil {
			b.WriteString(formatIssueComment(e.issueComment, r.ViewerLogin))
		} else {
			b.WriteString(formatReview(e.review, r.ViewerLogin))
		}
		if i < len(entries)-1 {
			b.WriteString("\n---\n")
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

func formatIssueComment(c *IssueComment, viewerLogin string) string {
	var b strings.Builder

	// header
	authorDisplay := formatAuthor(c.Author, viewerLogin)
	if c.IsMinimized {
		reason := c.MinimizedReason
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("issue #%d by %s | hidden: %s\n", c.DatabaseID, authorDisplay, reason))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("issue #%d by %s  \n", c.DatabaseID, authorDisplay))

	// date
	b.WriteString(fmt.Sprintf("_%s_\n", formatDate(c.CreatedAt)))
	b.WriteString("\n")

	// body
	b.WriteString(c.Body + "\n")
	b.WriteString("\n")

	// reactions
	b.WriteString(formatReactions(c.Reactions, viewerLogin))

	return b.String()
}

func formatReview(r *Review, viewerLogin string) string {
	var b strings.Builder

	// header
	authorDisplay := formatAuthor(r.Author, viewerLogin)

	// hidden review: resolved or dismissed
	if r.AllResolved {
		b.WriteString(fmt.Sprintf("review #%d by %s | hidden: Resolved\n", r.DatabaseID, authorDisplay))
		return b.String()
	}
	if r.State == "DISMISSED" {
		b.WriteString(fmt.Sprintf("review #%d by %s | hidden: Dismissed\n", r.DatabaseID, authorDisplay))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("review #%d by %s  \n", r.DatabaseID, authorDisplay))

	// date
	b.WriteString(fmt.Sprintf("_%s_\n", formatDate(r.CreatedAt)))
	b.WriteString("\n")

	// body
	if r.Body != "" {
		b.WriteString(r.Body + "\n")
		b.WriteString("\n")
	}

	// comment count
	if r.CommentCount > 0 {
		b.WriteString(fmt.Sprintf("comments: %d  \n", r.CommentCount))
	}

	// reactions
	b.WriteString(formatReactions(r.Reactions, viewerLogin))

	return b.String()
}

// isViewer checks if the login belongs to the authenticated user.
// Handles GitHub App bot suffix: viewer "my-app[bot]" matches author "my-app" and vice versa.
func isViewer(login, viewerLogin string) bool {
	return strings.TrimSuffix(login, "[bot]") == strings.TrimSuffix(viewerLogin, "[bot]")
}

// formatAuthor returns "you (@login)" if the author is the viewer, otherwise "@login"
func formatAuthor(login, viewerLogin string) string {
	if isViewer(login, viewerLogin) {
		return fmt.Sprintf("you (@%s)", login)
	}
	return "@" + login
}

// formatDate trims the ISO timestamp to "YYYY-MM-DD HH:MM:SS"
func formatDate(isoDate string) string {
	// input: "2026-03-11T12:15:54Z" → "2026-03-11 12:15:54"
	s := strings.TrimSuffix(isoDate, "Z")
	s = strings.Replace(s, "T", " ", 1)
	return s
}

// formatReactions renders reaction summary and "by you" line
func formatReactions(reactions []Reaction, viewerLogin string) string {
	if len(reactions) == 0 {
		return ""
	}

	// count reactions by type
	counts := make(map[string]int)
	var byViewer []string
	for _, r := range reactions {
		emoji := reactionEmoji[r.Content]
		if emoji == "" {
			emoji = r.Content
		}
		counts[emoji]++
		if isViewer(r.Author, viewerLogin) {
			byViewer = append(byViewer, emoji)
		}
	}

	var b strings.Builder

	// total reactions line: (1 🚀 2 👀)
	var parts []string
	for emoji, count := range counts {
		parts = append(parts, fmt.Sprintf("%d %s", count, emoji))
	}
	b.WriteString(fmt.Sprintf("(%s)  \n", strings.Join(parts, " ")))

	// reactions by you line
	if len(byViewer) > 0 {
		var viewerCounts = make(map[string]int)
		for _, e := range byViewer {
			viewerCounts[e]++
		}
		var viewerParts []string
		for emoji, count := range viewerCounts {
			viewerParts = append(viewerParts, fmt.Sprintf("%d %s", count, emoji))
		}
		b.WriteString(fmt.Sprintf("reactions by you: (%s)  \n", strings.Join(viewerParts, " ")))
	} else {
		b.WriteString("reactions by you:  \n")
	}

	return b.String()
}
