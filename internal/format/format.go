package format

import (
	"fmt"
	"strings"
)

// Reaction holds a single reaction with its author
type Reaction struct {
	Content string
	Author  string
}

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

// ReactionEmoji returns the emoji for a GraphQL reaction content enum, or the raw string.
func ReactionEmoji(content string) string {
	if e := reactionEmoji[content]; e != "" {
		return e
	}
	return content
}

// IsViewer checks if the login belongs to the authenticated user.
// Handles GitHub App bot suffix: viewer "my-app[bot]" matches author "my-app" and vice versa.
func IsViewer(login, viewerLogin string) bool {
	return strings.TrimSuffix(login, "[bot]") == strings.TrimSuffix(viewerLogin, "[bot]")
}

// Author returns "you (@login)" if the author is the viewer, otherwise "@login".
func Author(login, viewerLogin string) string {
	if IsViewer(login, viewerLogin) {
		return fmt.Sprintf("you (@%s)", login)
	}
	return "@" + login
}

// Date trims the ISO timestamp to "YYYY-MM-DD HH:MM:SS".
func Date(isoDate string) string {
	s := strings.TrimSuffix(isoDate, "Z")
	s = strings.Replace(s, "T", " ", 1)
	return s
}

// EnumLabel converts a GraphQL enum value (e.g. "OFF_TOPIC") to a readable label ("Off topic").
func EnumLabel(enum string) string {
	if enum == "" {
		return ""
	}
	s := strings.ReplaceAll(strings.ToLower(enum), "_", " ")
	return strings.ToUpper(s[:1]) + s[1:]
}

// Reactions renders reaction summary and "by you" line.
// Returns empty string if there are no reactions.
func Reactions(reactions []Reaction, viewerLogin string) string {
	if len(reactions) == 0 {
		return ""
	}

	counts := make(map[string]int)
	var byViewer []string
	for _, r := range reactions {
		emoji := ReactionEmoji(r.Content)
		counts[emoji]++
		if IsViewer(r.Author, viewerLogin) {
			byViewer = append(byViewer, emoji)
		}
	}

	var b strings.Builder

	var parts []string
	for emoji, count := range counts {
		parts = append(parts, fmt.Sprintf("%d %s", count, emoji))
	}
	b.WriteString(fmt.Sprintf("(%s)  \n", strings.Join(parts, " ")))

	if len(byViewer) > 0 {
		viewerCounts := make(map[string]int)
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
