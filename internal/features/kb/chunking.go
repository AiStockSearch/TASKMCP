package kb

import (
	"regexp"
	"strings"
)

type Chunked struct {
	ChunkIndex int               `json:"chunk_index"`
	Content    string            `json:"content"`
	Metadata   map[string]any    `json:"metadata"`
}

var headingRe = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)

type mdSection struct {
	headingPath []string
	heading     string
	level       int
	content     string
}

func chunkMarkdown(md string, maxChars int, overlapChars int) []Chunked {
	if maxChars <= 0 {
		maxChars = 3000
	}
	if overlapChars < 0 {
		overlapChars = 0
	}
	if overlapChars > maxChars/2 {
		overlapChars = maxChars / 2
	}

	sections := splitIntoSections(md)

	var out []Chunked
	chunkIndex := 0
	for _, s := range sections {
		sub := splitSectionContent(s.content, maxChars, overlapChars)
		for i := range sub {
			meta := map[string]any{
				"heading_path":   s.headingPath,
				"heading":        s.heading,
				"heading_level":  s.level,
				"chunk_subindex": i,
			}
			out = append(out, Chunked{
				ChunkIndex: chunkIndex,
				Content:    sub[i],
				Metadata:   meta,
			})
			chunkIndex++
		}
	}
	return out
}

func splitIntoSections(md string) []mdSection {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	lines := strings.Split(md, "\n")

	type stackItem struct {
		level int
		text  string
	}

	var (
		stack   []stackItem
		curr    mdSection
		builder strings.Builder
		out     []mdSection
	)

	flush := func() {
		curr.content = strings.TrimSpace(builder.String())
		if curr.heading == "" && len(curr.headingPath) == 0 {
			curr.heading = "Document"
			curr.level = 0
			curr.headingPath = []string{"Document"}
		}
		if curr.content != "" {
			out = append(out, curr)
		}
		builder.Reset()
	}

	for _, ln := range lines {
		if m := headingRe.FindStringSubmatch(ln); m != nil {
			// new section
			flush()

			level := len(m[1])
			title := strings.TrimSpace(m[2])

			// update stack (pop >= level)
			for len(stack) > 0 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, stackItem{level: level, text: title})

			path := make([]string, 0, len(stack))
			for _, it := range stack {
				path = append(path, it.text)
			}

			curr = mdSection{
				headingPath: path,
				heading:     title,
				level:       level,
			}

			// include the heading line in the content for stronger retrieval
			builder.WriteString(ln)
			builder.WriteString("\n")
			continue
		}

		builder.WriteString(ln)
		builder.WriteString("\n")
	}

	flush()
	return out
}

// splitSectionContent splits section text into chunks of ~maxChars with overlap, while trying
// not to cut inside fenced code blocks. It uses blank-line boundaries where possible.
func splitSectionContent(s string, maxChars int, overlapChars int) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if len(s) <= maxChars {
		return []string{s}
	}

	lines := strings.Split(s, "\n")
	var out []string

	inFence := false
	fenceDelim := ""

	startLine := 0
	for startLine < len(lines) {
		var b strings.Builder
		i := startLine
		lastSafeLine := -1

		for i < len(lines) {
			ln := lines[i]
			trim := strings.TrimSpace(ln)
			if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
				if !inFence {
					inFence = true
					fenceDelim = trim[:3]
				} else if fenceDelim != "" && strings.HasPrefix(trim, fenceDelim) {
					inFence = false
				}
			}

			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(ln)

			if !inFence && trim == "" {
				lastSafeLine = i
			}
			if b.Len() >= maxChars {
				break
			}
			i++
		}

		endLine := i
		if endLine >= len(lines) {
			out = append(out, strings.TrimSpace(b.String()))
			break
		}

		// Prefer cutting at last blank line, but only if it is after startLine.
		if lastSafeLine > startLine {
			endLine = lastSafeLine
		}

		chunk := strings.TrimSpace(strings.Join(lines[startLine:endLine+1], "\n"))
		if chunk != "" {
			out = append(out, chunk)
		}

		// Advance startLine with overlap by characters (approximate by walking backwards).
		if overlapChars <= 0 {
			startLine = endLine + 1
			continue
		}

		// compute new start line by counting chars backwards from endLine
		chars := 0
		j := endLine
		for j > startLine {
			chars += len(lines[j]) + 1
			if chars >= overlapChars {
				break
			}
			j--
		}
		startLine = j
		if startLine <= endLine {
			// ensure progress
			startLine = endLine + 1
		}
	}

	return out
}

