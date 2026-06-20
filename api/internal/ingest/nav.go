package ingest

import (
	"sort"
	"strings"
)

// NavNode is one entry in a source's navigation tree. Leaves carry a Slug;
// sections carry Children. A section may also have a Slug if a page lives at
// that path (e.g. an index page).
type NavNode struct {
	Title    string    `json:"title"`
	Slug     string    `json:"slug,omitempty"`
	Children []NavNode `json:"children,omitempty"`
}

// DocMeta is the minimal page info needed to build the navigation tree.
type DocMeta struct {
	Slug     string
	Title    string
	Position int
}

type navTmp struct {
	key      string
	title    string
	slug     string
	position int
	isLeaf   bool
	order    []string
	children map[string]*navTmp
}

const noPosition = 1 << 30

// BuildNav turns a flat list of pages into a nested navigation tree based on
// their slug path segments, ordered by position then title.
func BuildNav(docs []DocMeta) []NavNode {
	root := &navTmp{children: map[string]*navTmp{}, position: noPosition}

	for _, d := range docs {
		segs := strings.Split(d.Slug, "/")
		cur := root
		for i, seg := range segs {
			child, ok := cur.children[seg]
			if !ok {
				child = &navTmp{key: seg, children: map[string]*navTmp{}, position: noPosition}
				cur.children[seg] = child
				cur.order = append(cur.order, seg)
			}
			if i == len(segs)-1 {
				child.isLeaf = true
				child.title = d.Title
				child.slug = d.Slug
				if d.Position != 0 {
					child.position = d.Position
				}
			}
			cur = child
		}
	}

	return root.toNodes()
}

func (t *navTmp) toNodes() []NavNode {
	sort.SliceStable(t.order, func(i, j int) bool {
		a, b := t.children[t.order[i]], t.children[t.order[j]]
		if a.position != b.position {
			return a.position < b.position
		}
		return a.sortTitle() < b.sortTitle()
	})

	nodes := make([]NavNode, 0, len(t.order))
	for _, k := range t.order {
		c := t.children[k]
		n := NavNode{Title: c.displayTitle()}
		if c.isLeaf {
			n.Slug = c.slug
		}
		if len(c.children) > 0 {
			n.Children = c.toNodes()
		}
		nodes = append(nodes, n)
	}
	return nodes
}

func (t *navTmp) displayTitle() string {
	if t.isLeaf && t.title != "" {
		return t.title
	}
	return humanize(t.key)
}

func (t *navTmp) sortTitle() string {
	return strings.ToLower(t.displayTitle())
}

// humanize turns a slug segment into a readable title ("getting-started" ->
// "Getting Started").
func humanize(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
