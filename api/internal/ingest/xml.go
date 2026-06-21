package ingest

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	xmlCommentRe  = regexp.MustCompile(`(?s)<!--.*?-->`)
	xmlDoctypeRe  = regexp.MustCompile(`(?is)<!DOCTYPE.*?>`)
	xmlProgramRe  = regexp.MustCompile(`(?is)<(programlisting|screen|synopsis)[^>]*>(.*?)</(programlisting|screen|synopsis)>`)
	xmlRefTitleRe = regexp.MustCompile(`(?is)<refentrytitle[^>]*>(.*?)</refentrytitle>`)
	xmlRefNameRe  = regexp.MustCompile(`(?is)<refname[^>]*>(.*?)</refname>`)
	xmlTitleRe    = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	xmlBreakRe    = regexp.MustCompile(`(?is)</(para|simpara|section|sect1|sect2|sect3|refsect1|refsect2|refentry|chapter)>`)
	xmlTagRe      = regexp.MustCompile(`(?s)<[^>]+>`)
	xmlEntityRe   = regexp.MustCompile(`&[A-Za-z0-9_.-]+;`)
	xmlPunctRe    = regexp.MustCompile(`\s+([.,;:!?])`)
	xmlBlankRe    = regexp.MustCompile(`\n{3,}`)
)

// docBookXMLToMarkdown is a small fallback for DocBook-like manuals such as the
// PHP docs. It keeps text, titles, and code examples without trying to preserve
// the full source document semantics.
func docBookXMLToMarkdown(src string) string {
	s := normalizeNewlines(src)
	s = xmlCommentRe.ReplaceAllString(s, "")
	s = xmlDoctypeRe.ReplaceAllString(s, "")
	codeBlocks := make([]string, 0)
	s = xmlProgramRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := xmlTagRe.ReplaceAllString(match, "")
		codeBlocks = append(codeBlocks, cleanXMLCode(inner))
		return fmt.Sprintf("\n\n@@LORE_CODE_%d@@\n\n", len(codeBlocks)-1)
	})
	s = xmlTitleRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := xmlTagRe.ReplaceAllString(match, "")
		title := strings.TrimSpace(cleanXMLText(inner))
		if title == "" {
			return ""
		}
		return "\n\n## " + title + "\n\n"
	})
	s = xmlBreakRe.ReplaceAllString(s, "\n\n")
	s = xmlTagRe.ReplaceAllString(s, " ")
	s = cleanXMLText(s)
	for i, code := range codeBlocks {
		s = strings.ReplaceAll(s, fmt.Sprintf("@@LORE_CODE_%d@@", i), "\n\n```\n"+code+"\n```\n\n")
	}
	s = xmlBlankRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s) + "\n"
}

func docBookXMLTitle(src string) string {
	for _, re := range []*regexp.Regexp{xmlRefTitleRe, xmlRefNameRe, xmlTitleRe} {
		match := re.FindStringSubmatch(src)
		if len(match) < 2 {
			continue
		}
		title := strings.TrimSpace(cleanXMLText(xmlTagRe.ReplaceAllString(match[1], "")))
		if title != "" {
			return title
		}
	}
	return ""
}

func cleanXMLText(s string) string {
	s = html.UnescapeString(s)
	s = xmlEntityRe.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = xmlPunctRe.ReplaceAllString(strings.Join(strings.Fields(line), " "), "$1")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func cleanXMLCode(s string) string {
	s = html.UnescapeString(s)
	s = xmlEntityRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
