package comment

import (
	"fmt"
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

// Format renders the full comments result as human-readable markdown
func (r *CommentsResult) Format() string {
	var b strings.Builder

	// issue comments
	for i, c := range r.IssueComments {
		if i > 0 || len(r.Reviews) > 0 {
			// separator between entries is implicit from the header
		}
		b.WriteString(formatIssueComment(&c, r.ViewerLogin))
		b.WriteString("\n")
	}

	// reviews
	for _, rev := range r.Reviews {
		b.WriteString(formatReview(&rev, r.ViewerLogin))
		b.WriteString("\n")
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
		b.WriteString(fmt.Sprintf("# issue #%d by %s | hidden: %s\n", c.DatabaseID, authorDisplay, reason))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("# issue #%d by %s\n", c.DatabaseID, authorDisplay))

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
		b.WriteString(fmt.Sprintf("# review #%d by %s | hidden: Resolved\n", r.DatabaseID, authorDisplay))
		return b.String()
	}
	if r.State == "DISMISSED" {
		b.WriteString(fmt.Sprintf("# review #%d by %s | hidden: Dismissed\n", r.DatabaseID, authorDisplay))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("# review #%d by %s\n", r.DatabaseID, authorDisplay))

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
		b.WriteString(fmt.Sprintf("comments: %d\n", r.CommentCount))
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
	b.WriteString(fmt.Sprintf("(%s)\n", strings.Join(parts, " ")))

	// by you line
	if len(byViewer) > 0 {
		var viewerCounts = make(map[string]int)
		for _, e := range byViewer {
			viewerCounts[e]++
		}
		var viewerParts []string
		for emoji, count := range viewerCounts {
			viewerParts = append(viewerParts, fmt.Sprintf("%d %s", count, emoji))
		}
		b.WriteString(fmt.Sprintf("by you: (%s)\n", strings.Join(viewerParts, " ")))
	} else {
		b.WriteString("by you:\n")
	}

	return b.String()
}
