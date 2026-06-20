package ingest

import (
	"strings"
	"testing"
)

func TestDeriveSlug(t *testing.T) {
	cases := map[string]string{
		"getting-started.mdx":           "getting-started",
		"concepts/why-astro.mdx":        "concepts/why-astro",
		"concepts/index.mdx":            "concepts",
		"(index).mdx":                   "index",
		"01-introduction.md":            "introduction",
		"guides/01-deploy/02-vercel.md": "guides/deploy/vercel",
		"README.md":                     "index",
		"API Reference/Foo.md":          "api-reference/foo",
	}
	for in, want := range cases {
		if got := DeriveSlug(in); got != want {
			t.Errorf("DeriveSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSanitizeMDX(t *testing.T) {
	src := strings.Join([]string{
		`import { Foo } from "@/components/foo";`,
		`export const title = "My Title";`,
		`export const description = "desc";`,
		``,
		`# Heading`,
		``,
		`Some text.`,
		``,
		`<Foo bar={1} />`,
		``,
		`More text.`,
		``,
		`<Tabs>`,
		`  <TabItem>hidden</TabItem>`,
		`</Tabs>`,
		``,
		"```go",
		`<NotJSXBecauseInCode />`,
		"```",
		``,
		`End.`,
	}, "\n")

	clean, title, desc, warnings := sanitizeMDX(src)

	if title != "My Title" {
		t.Errorf("title = %q, want %q", title, "My Title")
	}
	if desc != "desc" {
		t.Errorf("desc = %q, want %q", desc, "desc")
	}
	if warnings < 2 {
		t.Errorf("warnings = %d, want >= 2", warnings)
	}

	mustNotContain := []string{"import", "export const", "<Foo", "<Tabs>", "TabItem"}
	for _, s := range mustNotContain {
		if strings.Contains(clean, s) {
			t.Errorf("clean still contains %q:\n%s", s, clean)
		}
	}
	mustContain := []string{"# Heading", "Some text.", "More text.", "End.", "<NotJSXBecauseInCode />"}
	for _, s := range mustContain {
		if !strings.Contains(clean, s) {
			t.Errorf("clean missing %q:\n%s", s, clean)
		}
	}
}

func TestBuildNav(t *testing.T) {
	docs := []DocMeta{
		{Slug: "getting-started", Title: "Getting started", Position: 1},
		{Slug: "concepts/why-astro", Title: "Why Astro", Position: 0},
		{Slug: "concepts", Title: "Concepts", Position: 0},
	}
	nav := BuildNav(docs)

	var concepts *NavNode
	for i := range nav {
		if nav[i].Slug == "concepts" || nav[i].Title == "Concepts" {
			concepts = &nav[i]
		}
	}
	if concepts == nil {
		t.Fatalf("no concepts node in nav: %+v", nav)
	}
	if len(concepts.Children) != 1 || concepts.Children[0].Slug != "concepts/why-astro" {
		t.Fatalf("concepts children = %+v, want one child concepts/why-astro", concepts.Children)
	}
}

func TestRender(t *testing.T) {
	r := NewRenderer()
	md := strings.Join([]string{
		`## Alpha`,
		``,
		`Some **bold** text with ` + "`code`" + `.`,
		``,
		`### Beta`,
		``,
		"```go",
		`func main() { println("hi") }`,
		"```",
		``,
		`<script>alert(1)</script>`,
	}, "\n")

	out, err := r.Render([]byte(md))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(out.HTML, "<script") {
		t.Errorf("script tag not sanitized:\n%s", out.HTML)
	}
	if !strings.Contains(out.HTML, "<pre") {
		t.Errorf("expected a code block in HTML:\n%s", out.HTML)
	}
	if !strings.Contains(out.HTML, "chroma") {
		t.Errorf("expected chroma highlight classes in HTML:\n%s", out.HTML)
	}
	if len(out.TOC) != 1 || out.TOC[0].Title != "Alpha" {
		t.Fatalf("TOC = %+v, want one H2 'Alpha'", out.TOC)
	}
	if out.TOC[0].Anchor == "" {
		t.Errorf("expected non-empty anchor for H2")
	}
	if len(out.TOC[0].Children) != 1 || out.TOC[0].Children[0].Title != "Beta" {
		t.Fatalf("TOC children = %+v, want one H3 'Beta'", out.TOC[0].Children)
	}
	if !strings.Contains(out.Text, "bold") || strings.Contains(out.Text, "<") {
		t.Errorf("Text not clean plain text: %q", out.Text)
	}
}

func TestRSTToMarkdown(t *testing.T) {
	src := strings.Join([]string{
		"Python Tutorial",
		"===============",
		"",
		"Use :func:`print` and ``len``.",
		"",
		"Example::",
		"",
		"    print('hello')",
	}, "\n")

	got := rstToMarkdown(src)
	for _, want := range []string{"# Python Tutorial", "`print`", "`len`", "```", "print('hello')"} {
		if !strings.Contains(got, want) {
			t.Fatalf("rstToMarkdown missing %q:\n%s", want, got)
		}
	}
}

func TestDocBookXMLToMarkdown(t *testing.T) {
	src := `<section><title>Basic syntax</title><para>Hello <function>echo</function>.</para><programlisting>&lt;?php echo "hi"; ?&gt;</programlisting></section>`

	got := docBookXMLToMarkdown(src)
	for _, want := range []string{"## Basic syntax", "Hello echo.", "```", `<?php echo "hi"; ?>`} {
		if !strings.Contains(got, want) {
			t.Fatalf("docBookXMLToMarkdown missing %q:\n%s", want, got)
		}
	}
}
