package review

import (
	"fmt"
	"strings"
)

// Format renders the review detail as human-readable markdown
func (d *ReviewDetail) Format() string {
	var b strings.Builder

	// header
	authorDisplay := formatAuthor(d.Author, d.ViewerLogin)
	b.WriteString(fmt.Sprintf("review #%d by %s  \n", d.DatabaseID, authorDisplay))
	b.WriteString(fmt.Sprintf("_%s_\n", formatDate(d.CreatedAt)))
	b.WriteString("\n")

	// body
	if d.Body != "" {
		b.WriteString(d.Body + "\n")
		b.WriteString("\n")
	}

	// group comments into threads: top-level comments and their replies
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
// A thread is a top-level comment (ReplyToID == 0) followed by its replies.
func buildThreads(comments []ReviewComment) []commentThread {
	// index top-level comments by ID
	threadMap := make(map[int64]*commentThread)
	var threadOrder []int64

	for _, c := range comments {
		if c.ReplyToID == 0 {
			threadMap[c.DatabaseID] = &commentThread{root: c}
			threadOrder = append(threadOrder, c.DatabaseID)
		}
	}

	// assign replies to their parent threads
	for _, c := range comments {
		if c.ReplyToID != 0 {
			if thread, ok := threadMap[c.ReplyToID]; ok {
				thread.replies = append(thread.replies, c)
			} else {
				// reply to a reply — find the root thread by walking up
				// For simplicity, just find any thread that contains the parent
				placed := false
				for _, tid := range threadOrder {
					t := threadMap[tid]
					if t.root.DatabaseID == c.ReplyToID {
						t.replies = append(t.replies, c)
						placed = true
						break
					}
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
		}
	}

	// collect threads in order
	var threads []commentThread
	for _, id := range threadOrder {
		threads = append(threads, *threadMap[id])
	}
	return threads
}

func formatThread(thread commentThread, viewerLogin string) string {
	var b strings.Builder

	// root comment with file context
	b.WriteString(formatReviewComment(&thread.root, viewerLogin, false))

	// replies indented with >
	for _, reply := range thread.replies {
		b.WriteString(formatReviewComment(&reply, viewerLogin, true))
	}

	return b.String()
}

func formatReviewComment(c *ReviewComment, viewerLogin string, isReply bool) string {
	var b strings.Builder

	authorDisplay := formatAuthor(c.Author, viewerLogin)
	prefix := ""
	if isReply {
		prefix = "> "
	}

	if c.IsMinimized {
		reason := c.MinimizedReason
		if reason == "" {
			reason = "hidden"
		}
		b.WriteString(fmt.Sprintf("%s**%s** | hidden: %s\n", prefix, authorDisplay, reason))
		return b.String()
	}

	// header line with file context
	if !isReply && c.Path != "" {
		location := c.Path
		if c.Line > 0 {
			location = fmt.Sprintf("%s#%d", c.Path, c.Line)
		}
		outdatedMark := ""
		if c.Outdated {
			outdatedMark = " (outdated)"
		}
		b.WriteString(fmt.Sprintf("%s**%s** %s%s  \n", prefix, authorDisplay, location, outdatedMark))
	} else {
		b.WriteString(fmt.Sprintf("%s**%s**  \n", prefix, authorDisplay))
	}

	// date
	b.WriteString(fmt.Sprintf("%s_%s_\n", prefix, formatDate(c.CreatedAt)))

	// diff hunk for top-level comments
	if !isReply && c.DiffHunk != "" {
		b.WriteString(fmt.Sprintf("%s```diff\n", prefix))
		for _, line := range strings.Split(c.DiffHunk, "\n") {
			b.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
		}
		b.WriteString(fmt.Sprintf("%s```\n", prefix))
	}

	// body
	for _, line := range strings.Split(c.Body, "\n") {
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
	}

	return b.String()
}

// isViewer checks if the login belongs to the authenticated user.
func isViewer(login, viewerLogin string) bool {
	return strings.TrimSuffix(login, "[bot]") == strings.TrimSuffix(viewerLogin, "[bot]")
}

func formatAuthor(login, viewerLogin string) string {
	if isViewer(login, viewerLogin) {
		return fmt.Sprintf("you (@%s)", login)
	}
	return "@" + login
}

func formatDate(isoDate string) string {
	s := strings.TrimSuffix(isoDate, "Z")
	s = strings.Replace(s, "T", " ", 1)
	return s
}
