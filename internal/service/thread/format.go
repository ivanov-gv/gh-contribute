package thread

import (
	"fmt"
	"strings"

	"github.com/ivanov-gv/gh-contribute/internal/utils/format"
)

// Format renders the thread as human-readable markdown.
func (t *Thread) Format() string {
	var b strings.Builder

	location := formatLocation(t)
	b.WriteString(fmt.Sprintf("# thread #%d  %s  \n", t.ThreadID, location))
	b.WriteString("\n")

	for i, c := range t.Comments {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		b.WriteString(formatThreadComment(c, t.ViewerLogin))
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

func formatThreadComment(c ThreadComment, viewerLogin string) string {
	var b strings.Builder
	authorDisplay := format.Author(c.Author, viewerLogin)

	if c.IsMinimized {
		reason := format.EnumLabel(c.MinimizedReason)
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("comment #%d by %s  review #%d | hidden: %s\n", c.DatabaseID, authorDisplay, c.ReviewDatabaseID, reason))
		return b.String()
	}

	if c.ReplyToID == 0 {
		b.WriteString(fmt.Sprintf("comment #%d by %s  review #%d  \n", c.DatabaseID, authorDisplay, c.ReviewDatabaseID))
	} else {
		b.WriteString(fmt.Sprintf("reply #%d to #%d  by %s  review #%d  \n", c.DatabaseID, c.ReplyToID, authorDisplay, c.ReviewDatabaseID))
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
func formatLocation(t *Thread) string {
	if t.Path == "" {
		return ""
	}
	if t.IsOutdated {
		return formatOutdatedLocation(t)
	}
	return formatCurrentLocation(t)
}

func formatCurrentLocation(t *Thread) string {
	if t.StartLine > 0 && t.Line > 0 && t.StartLine != t.Line {
		return fmt.Sprintf("%s on lines +%d to +%d", t.Path, t.StartLine, t.Line)
	} else if t.Line > 0 {
		return fmt.Sprintf("%s on line +%d", t.Path, t.Line)
	}
	return t.Path
}

func formatOutdatedLocation(t *Thread) string {
	if t.OriginalStartLine > 0 && t.OriginalLine > 0 && t.OriginalStartLine != t.OriginalLine {
		return fmt.Sprintf("%s on original lines %d to %d (outdated)", t.Path, t.OriginalStartLine, t.OriginalLine)
	} else if t.OriginalLine > 0 {
		return fmt.Sprintf("%s on original line %d (outdated)", t.Path, t.OriginalLine)
	}
	return fmt.Sprintf("%s (outdated)", t.Path)
}
