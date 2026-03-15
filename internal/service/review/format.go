package review

import (
	"fmt"
	"strings"

	"github.com/ivanov-gv/gh-contribute/internal/utils/format"
)

// Format renders the review detail as human-readable markdown.
// When showDiff is true, the diff hunk is included below each thread header.
func (d *ReviewDetail) Format(showDiff bool) string {
	var b strings.Builder

	authorDisplay := format.Author(d.Author, d.ViewerLogin)
	b.WriteString(fmt.Sprintf("# review #%d by %s  \n", d.DatabaseID, authorDisplay))
	b.WriteString(fmt.Sprintf("_%s_\n", format.Date(d.CreatedAt)))
	b.WriteString("\n")

	if d.Body != "" {
		b.WriteString(d.Body + "\n")
		b.WriteString("\n")
	}

	b.WriteString(format.Reactions(d.Reactions, d.ViewerLogin))

	for i, group := range d.ThreadGroups {
		if i > 0 {
			b.WriteString("\n---\n")
		} else {
			b.WriteString("\n")
		}
		b.WriteString(formatThreadGroup(group, d.ViewerLogin, showDiff))
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

func formatThreadGroup(g ReviewThreadGroup, viewerLogin string, showDiff bool) string {
	var b strings.Builder

	// thread header: ID and location so the user can look up the full thread
	location := formatLocation(g)
	b.WriteString(fmt.Sprintf("thread #%d  %s  \n", g.ThreadID, location))

	if showDiff && g.DiffHunk != "" {
		b.WriteString("\n```diff\n")
		b.WriteString(g.DiffHunk + "\n")
		b.WriteString("```\n")
	}

	for _, c := range g.Comments {
		b.WriteString(formatReviewComment(c, viewerLogin))
	}

	return b.String()
}

func formatReviewComment(c ReviewComment, viewerLogin string) string {
	var b strings.Builder
	authorDisplay := format.Author(c.Author, viewerLogin)

	if c.IsMinimized {
		reason := format.EnumLabel(c.MinimizedReason)
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("comment #%d by %s | hidden: %s\n", c.DatabaseID, authorDisplay, reason))
		return b.String()
	}

	if c.ReplyToID == 0 {
		// thread root
		b.WriteString(fmt.Sprintf("comment #%d by %s  \n", c.DatabaseID, authorDisplay))
	} else if c.ReplyToIsExternal {
		// reply to a comment from another review — flag it clearly
		b.WriteString(fmt.Sprintf("reply #%d to #%d (not in this review)  by %s  \n", c.DatabaseID, c.ReplyToID, authorDisplay))
	} else {
		b.WriteString(fmt.Sprintf("reply #%d to #%d  by %s  \n", c.DatabaseID, c.ReplyToID, authorDisplay))
	}

	b.WriteString(fmt.Sprintf("_%s_\n", format.Date(c.CreatedAt)))
	b.WriteString("\n")

	for _, line := range strings.Split(c.Body, "\n") {
		b.WriteString(line + "\n")
	}

	if reactionsStr := format.Reactions(c.Reactions, viewerLogin); reactionsStr != "" {
		b.WriteString(reactionsStr)
	}

	return b.String()
}

// formatLocation builds the location string from thread-level fields.
// For up-to-date: `path on lines +startLine to +line`
// For outdated:   `path on original lines startLine to line (outdated)`
func formatLocation(g ReviewThreadGroup) string {
	if g.Path == "" {
		return ""
	}
	if g.IsOutdated {
		return formatOutdatedLocation(g)
	}
	return formatCurrentLocation(g)
}

func formatCurrentLocation(g ReviewThreadGroup) string {
	if g.StartLine > 0 && g.Line > 0 && g.StartLine != g.Line {
		return fmt.Sprintf("%s on lines +%d to +%d", g.Path, g.StartLine, g.Line)
	} else if g.Line > 0 {
		return fmt.Sprintf("%s on line +%d", g.Path, g.Line)
	}
	return g.Path
}

func formatOutdatedLocation(g ReviewThreadGroup) string {
	if g.OriginalStartLine > 0 && g.OriginalLine > 0 && g.OriginalStartLine != g.OriginalLine {
		return fmt.Sprintf("%s on original lines %d to %d (outdated)", g.Path, g.OriginalStartLine, g.OriginalLine)
	} else if g.OriginalLine > 0 {
		return fmt.Sprintf("%s on original line %d (outdated)", g.Path, g.OriginalLine)
	}
	return fmt.Sprintf("%s (outdated)", g.Path)
}
