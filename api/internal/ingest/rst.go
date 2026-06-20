package ingest

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	rstRoleRe      = regexp.MustCompile(":[A-Za-z0-9_-]+:`([^`]+)`")
	rstLinkRe      = regexp.MustCompile("`([^`<]+)\\s*<[^`]+>`_")
	rstInlineCode  = regexp.MustCompile("``([^`]+)``")
	rstUnknownLink = regexp.MustCompile("`([^`]+)`_")
)

// rstToMarkdown performs a conservative, lossy conversion for official docs
// that are authored in reStructuredText (notably Python). It preserves headings,
// paragraphs, and common code blocks so the existing Markdown renderer can index
// and display the page.
func rstToMarkdown(src string) string {
	lines := strings.Split(normalizeNewlines(src), "\n")
	out := make([]string, 0, len(lines))

	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], " \t")
		trimmed := strings.TrimSpace(line)

		if i+1 < len(lines) && isRSTHeadingUnderline(lines[i+1], trimmed) {
			out = append(out, strings.Repeat("#", rstHeadingLevel(lines[i+1]))+" "+cleanRSTInline(trimmed))
			i++
			continue
		}

		if strings.HasPrefix(trimmed, ".. _") || strings.HasPrefix(trimmed, ".. |") {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, ".. toctree::"),
			strings.HasPrefix(trimmed, ".. contents::"),
			strings.HasPrefix(trimmed, ".. module::"),
			strings.HasPrefix(trimmed, ".. currentmodule::"):
			i = skipIndentedDirective(lines, i)
			continue
		case strings.HasPrefix(trimmed, ".. code-block::"),
			strings.HasPrefix(trimmed, ".. sourcecode::"),
			strings.HasPrefix(trimmed, ".. doctest::"):
			lang := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(trimmed, ".. code-block::"), ".. sourcecode::"), ".. doctest::"))
			block, next := collectRSTIndentedBlock(lines, i+1)
			out = append(out, "", "```"+lang)
			out = append(out, block...)
			out = append(out, "```", "")
			i = next - 1
			continue
		case strings.HasPrefix(trimmed, ".. note::"):
			note := strings.TrimSpace(strings.TrimPrefix(trimmed, ".. note::"))
			if note != "" {
				out = append(out, "> **Note:** "+cleanRSTInline(note))
			}
			continue
		case strings.HasSuffix(trimmed, "::"):
			out = append(out, cleanRSTInline(strings.TrimSuffix(line, "::"))+":")
			block, next := collectRSTIndentedBlock(lines, i+1)
			if len(block) > 0 {
				out = append(out, "", "```")
				out = append(out, block...)
				out = append(out, "```", "")
				i = next - 1
			}
			continue
		}

		out = append(out, cleanRSTInline(line))
	}

	return strings.Join(out, "\n")
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func isRSTHeadingUnderline(line, title string) bool {
	if title == "" {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if utf8.RuneCountInString(trimmed) < 3 {
		return false
	}
	first, _ := utf8.DecodeRuneInString(trimmed)
	if !strings.ContainsRune("=-~^\"'", first) {
		return false
	}
	for _, r := range trimmed {
		if r != first {
			return false
		}
	}
	return utf8.RuneCountInString(trimmed) >= utf8.RuneCountInString(title)
}

func rstHeadingLevel(line string) int {
	switch strings.TrimSpace(line)[0] {
	case '=':
		return 1
	case '-':
		return 2
	case '~':
		return 3
	case '^':
		return 4
	default:
		return 5
	}
}

func cleanRSTInline(s string) string {
	s = rstRoleRe.ReplaceAllString(s, "`$1`")
	s = strings.ReplaceAll(s, "`!", "`")
	s = rstLinkRe.ReplaceAllString(s, "$1")
	s = rstInlineCode.ReplaceAllString(s, "`$1`")
	s = rstUnknownLink.ReplaceAllString(s, "$1")
	return s
}

func skipIndentedDirective(lines []string, index int) int {
	_, next := collectRSTIndentedBlock(lines, index+1)
	return next - 1
}

func collectRSTIndentedBlock(lines []string, start int) ([]string, int) {
	i := start
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ":") {
		i++
	}

	block := make([]string, 0)
	for i < len(lines) {
		line := strings.TrimRight(lines[i], " \t")
		if strings.TrimSpace(line) == "" {
			block = append(block, "")
			i++
			continue
		}
		if leadingIndent(line) == 0 {
			break
		}
		block = append(block, line)
		i++
	}

	return stripCommonIndent(block), i
}

func stripCommonIndent(lines []string) []string {
	minIndent := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingIndent(line)
		if minIndent == 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent == 0 {
		return lines
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		if len(line) >= minIndent {
			out[i] = line[minIndent:]
		} else {
			out[i] = strings.TrimLeft(line, " \t")
		}
	}
	return out
}

func leadingIndent(s string) int {
	count := 0
	for _, r := range s {
		if r != ' ' && r != '\t' {
			break
		}
		count++
	}
	return count
}
