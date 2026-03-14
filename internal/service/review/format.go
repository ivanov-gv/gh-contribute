package review

import (
	"fmt"
	"strings"

	"github.com/ivanov-gv/gh-contribute/internal/format"
)

// Format renders the review detail as human-readable markdown
func (d *ReviewDetail) Format() string {
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

	if len(d.Comments) > 0 {
		b.WriteString("---\n")
	}

	threads := buildThreads(d.Comments)
	for i, thread := range threads {
		if i > 0 {
			b.WriteString("\n---\n")
		}
		b.WriteString(formatThread(thread, d.ViewerLogin))
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

func formatThread(thread commentThread, viewerLogin string) string {
	var b strings.Builder
	b.WriteString(formatReviewComment(&thread.root, viewerLogin, false))
	for _, reply := range thread.replies {
		b.WriteString(formatReviewComment(&reply, viewerLogin, true))
	}
	return b.String()
}

func formatReviewComment(c *ReviewComment, viewerLogin string, isReply bool) string {
	var b strings.Builder

	authorDisplay := format.Author(c.Author, viewerLogin)
	prefix := ""
	if isReply {
		prefix = "> "
	}

	if c.IsMinimized {
		reason := format.EnumLabel(c.MinimizedReason)
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("%scomment #%d by %s | hidden: %s\n", prefix, c.DatabaseID, authorDisplay, reason))
		return b.String()
	}

	if !isReply && c.Path != "" {
		location := c.Path
		if c.StartLine > 0 && c.Line > 0 && c.StartLine != c.Line {
			location = fmt.Sprintf("%s#%d-%d", c.Path, c.StartLine, c.Line)
		} else if c.Line > 0 {
			location = fmt.Sprintf("%s#%d", c.Path, c.Line)
		}
		outdatedMark := ""
		if c.Outdated {
			outdatedMark = " (outdated)"
		}
		b.WriteString(fmt.Sprintf("%scomment #%d by %s %s%s  \n", prefix, c.DatabaseID, authorDisplay, location, outdatedMark))
	} else {
		b.WriteString(fmt.Sprintf("%scomment #%d by %s  \n", prefix, c.DatabaseID, authorDisplay))
	}

	b.WriteString(fmt.Sprintf("%s_%s_\n", prefix, format.Date(c.CreatedAt)))
	b.WriteString(prefix + "\n")

	for _, line := range strings.Split(c.Body, "\n") {
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
	}

	reactionsStr := format.Reactions(c.Reactions, viewerLogin)
	if reactionsStr != "" {
		for _, line := range strings.Split(strings.TrimRight(reactionsStr, "\n"), "\n") {
			b.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
		}
	}

	return b.String()
}
