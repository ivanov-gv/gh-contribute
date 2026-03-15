package review

import (
	"fmt"
	"strings"

	"github.com/ivanov-gv/gh-contribute/internal/utils/format"
)

// Format renders the review detail as human-readable markdown.
// When showDiff is true, diffHunk is included for each comment.
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

	threads := buildThreads(d.Comments)
	for i, thread := range threads {
		b.WriteString("\n---\n")
		// thread numbering is 1-indexed sequential
		b.WriteString(formatThread(thread, i+1, d.ViewerLogin, showDiff))
		b.WriteString("\n---")
	}

	if len(threads) > 0 {
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// commentThread holds a top-level comment and its replies
type commentThread struct {
	root    ReviewComment
	replies []ReviewComment
}

// buildThreads groups comments into threads. Comments are already sorted by date.
func buildThreads(comments []ReviewComment) []commentThread {
	threadMap := make(map[int64]*commentThread)
	var threadOrder []int64

	for _, c := range comments {
		if c.ReplyToID == 0 {
			threadMap[c.DatabaseID] = &commentThread{root: c}
			threadOrder = append(threadOrder, c.DatabaseID)
		}
	}

	for _, c := range comments {
		if c.ReplyToID == 0 {
			continue
		}
		if thread, ok := threadMap[c.ReplyToID]; ok {
			thread.replies = append(thread.replies, c)
			continue
		}
		// reply to a reply — find the root thread by walking up
		placed := false
		for _, tid := range threadOrder {
			t := threadMap[tid]
			for _, r := range t.replies {
				if r.DatabaseID == c.ReplyToID {
					t.replies = append(t.replies, c)
					placed = true
					break
				}
			}
			if placed {
				break
			}
		}
		if !placed {
			// orphan reply — treat as top-level
			threadMap[c.DatabaseID] = &commentThread{root: c}
			threadOrder = append(threadOrder, c.DatabaseID)
		}
	}

	threads := make([]commentThread, len(threadOrder))
	for i, id := range threadOrder {
		threads[i] = *threadMap[id]
	}
	return threads
}

func formatThread(thread commentThread, threadNum int, viewerLogin string, showDiff bool) string {
	var b strings.Builder
	b.WriteString(formatThreadRoot(&thread.root, threadNum, viewerLogin, showDiff))
	for _, reply := range thread.replies {
		b.WriteString(formatReply(&reply, viewerLogin, showDiff))
	}
	return b.String()
}

// formatThreadRoot formats the root comment of a thread.
// Format: `thread #<N> comment #<ID> by <author>  <location>  `
func formatThreadRoot(c *ReviewComment, threadNum int, viewerLogin string, showDiff bool) string {
	var b strings.Builder
	authorDisplay := format.Author(c.Author, viewerLogin)

	if c.IsMinimized {
		reason := format.EnumLabel(c.MinimizedReason)
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("thread #%d comment #%d by %s | hidden: %s\n", threadNum, c.DatabaseID, authorDisplay, reason))
		return b.String()
	}

	location := formatLocation(c)
	b.WriteString(fmt.Sprintf("thread #%d comment #%d by %s  %s  \n", threadNum, c.DatabaseID, authorDisplay, location))
	b.WriteString(fmt.Sprintf("_%s_\n", format.Date(c.CreatedAt)))
	b.WriteString("\n")

	for _, line := range strings.Split(c.Body, "\n") {
		b.WriteString(line + "\n")
	}

	if showDiff && c.DiffHunk != "" {
		b.WriteString("\n```diff\n")
		b.WriteString(c.DiffHunk + "\n")
		b.WriteString("```\n")
	}

	reactionsStr := format.Reactions(c.Reactions, viewerLogin)
	if reactionsStr != "" {
		b.WriteString(reactionsStr)
	}

	return b.String()
}

// formatReply formats a reply comment in a thread.
// Format: `reply #<ID> to #<parentID>  by <author>`
func formatReply(c *ReviewComment, viewerLogin string, showDiff bool) string {
	var b strings.Builder
	authorDisplay := format.Author(c.Author, viewerLogin)

	if c.IsMinimized {
		reason := format.EnumLabel(c.MinimizedReason)
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("reply #%d to #%d  by %s | hidden: %s\n", c.DatabaseID, c.ReplyToID, authorDisplay, reason))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("reply #%d to #%d  by %s\n", c.DatabaseID, c.ReplyToID, authorDisplay))
	b.WriteString(fmt.Sprintf("_%s_\n", format.Date(c.CreatedAt)))
	b.WriteString("\n")

	for _, line := range strings.Split(c.Body, "\n") {
		b.WriteString(line + "\n")
	}

	if showDiff && c.DiffHunk != "" {
		b.WriteString("\n```diff\n")
		b.WriteString(c.DiffHunk + "\n")
		b.WriteString("```\n")
	}

	reactionsStr := format.Reactions(c.Reactions, viewerLogin)
	if reactionsStr != "" {
		b.WriteString(reactionsStr)
	}

	return b.String()
}

// formatLocation builds the file/line/commit location string for a comment header.
// For up-to-date: `path on lines +startLine to +line  commit <sha7>`
// For outdated:   `path on original lines startLine to line  original commit <sha7> (outdated)`
func formatLocation(c *ReviewComment) string {
	if c.Path == "" {
		return ""
	}

	if c.Outdated {
		return formatOutdatedLocation(c)
	}
	return formatCurrentLocation(c)
}

func formatCurrentLocation(c *ReviewComment) string {
	if c.StartLine > 0 && c.Line > 0 && c.StartLine != c.Line {
		return fmt.Sprintf("%s on lines +%d to +%d", c.Path, c.StartLine, c.Line)
	} else if c.Line > 0 {
		return fmt.Sprintf("%s on line +%d", c.Path, c.Line)
	}
	return c.Path
}

func formatOutdatedLocation(c *ReviewComment) string {
	if c.OriginalStartLine > 0 && c.OriginalLine > 0 && c.OriginalStartLine != c.OriginalLine {
		return fmt.Sprintf("%s on original lines %d to %d (outdated)", c.Path, c.OriginalStartLine, c.OriginalLine)
	} else if c.OriginalLine > 0 {
		return fmt.Sprintf("%s on original line %d (outdated)", c.Path, c.OriginalLine)
	}
	return fmt.Sprintf("%s (outdated)", c.Path)
}
