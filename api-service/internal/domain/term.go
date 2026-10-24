package domain

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

type TermKind string

const (
	TermRole     TermKind = "role"
	TermResource TermKind = "resource"
	TermAction   TermKind = "action"
)

func (k TermKind) Valid() bool { return k == TermRole || k == TermResource || k == TermAction }

// Term là 1 mục trong từ điển của project: tên chuẩn + các cách gọi khác.
type Term struct {
	ID          string
	ProjectID   string
	Kind        TermKind
	Canonical   string
	Synonyms    []string
	Description *string
	CreatedAt   time.Time
}

func (t *Term) Normalize() error {
	if !t.Kind.Valid() {
		return Validation("invalid_term_kind", "Kind must be role, resource or action")
	}
	t.Canonical = strings.TrimSpace(t.Canonical)
	if t.Canonical == "" {
		return Validation("term_canonical_required", "Canonical name is required")
	}
	seen := map[string]bool{strings.ToLower(t.Canonical): true}
	clean := []string{}
	for _, s := range t.Synonyms {
		s = strings.TrimSpace(s)
		if s == "" || seen[strings.ToLower(s)] {
			continue
		}
		seen[strings.ToLower(s)] = true
		clean = append(clean, s)
	}
	t.Synonyms = clean
	return nil
}

// TermNormalizer thay các cách gọi khác bằng tên chuẩn trước khi phân tích.
type TermNormalizer struct {
	canonical map[TermKind]map[string]string // synonym (lower) → canonical
	textRules []textRule
}

type textRule struct {
	re          *regexp.Regexp
	replacement string
}

func NewTermNormalizer(terms []Term) *TermNormalizer {
	n := &TermNormalizer{canonical: map[TermKind]map[string]string{}}
	type pair struct{ from, to string }
	var pairs []pair
	for _, t := range terms {
		if n.canonical[t.Kind] == nil {
			n.canonical[t.Kind] = map[string]string{}
		}
		n.canonical[t.Kind][strings.ToLower(t.Canonical)] = t.Canonical
		for _, s := range t.Synonyms {
			n.canonical[t.Kind][strings.ToLower(s)] = t.Canonical
			pairs = append(pairs, pair{s, t.Canonical})
		}
	}
	// Cụm dài thay trước để "sales employee" không bị "employee" thay mất
	sort.Slice(pairs, func(i, j int) bool { return len(pairs[i].from) > len(pairs[j].from) })
	for _, p := range pairs {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(p.from) + `\b`)
		n.textRules = append(n.textRules, textRule{re: re, replacement: p.to})
	}
	return n
}

// Canonical trả về tên chuẩn của value nếu có trong từ điển.
func (n *TermNormalizer) Canonical(kind TermKind, value string) string {
	if c, ok := n.canonical[kind][strings.ToLower(strings.TrimSpace(value))]; ok {
		return c
	}
	return value
}

func (n *TermNormalizer) Known(kind TermKind, value string) bool {
	_, ok := n.canonical[kind][strings.ToLower(strings.TrimSpace(value))]
	return ok
}

func (n *TermNormalizer) HasKind(kind TermKind) bool { return len(n.canonical[kind]) > 0 }

func (n *TermNormalizer) NormalizeText(s string) string {
	for _, r := range n.textRules {
		s = r.re.ReplaceAllLiteralString(s, r.replacement)
	}
	return s
}

func (n *TermNormalizer) NormalizeRow(r PMRow) PMRow {
	r.Role = n.Canonical(TermRole, r.Role)
	r.Action = n.Canonical(TermAction, r.Action)
	r.Resource = n.Canonical(TermResource, r.Resource)
	return r
}

// TermSuggestion: giá trị xuất hiện trong PM nhưng chưa có trong từ điển.
type TermSuggestion struct {
	Kind  TermKind
	Value string
	Count int
}

func SuggestTerms(n *TermNormalizer, rows []PMRow) []TermSuggestion {
	counts := map[TermKind]map[string]int{}
	display := map[string]string{}
	add := func(kind TermKind, v string) {
		v = strings.TrimSpace(v)
		if v == "" || n.Known(kind, v) {
			return
		}
		if counts[kind] == nil {
			counts[kind] = map[string]int{}
		}
		key := strings.ToLower(v)
		counts[kind][key]++
		if _, ok := display[string(kind)+key]; !ok {
			display[string(kind)+key] = v // giữ cách viết của lần xuất hiện đầu tiên
		}
	}
	for _, r := range rows {
		add(TermRole, r.Role)
		add(TermAction, r.Action)
		add(TermResource, r.Resource)
	}
	out := []TermSuggestion{}
	for _, kind := range []TermKind{TermRole, TermResource, TermAction} {
		for key, c := range counts[kind] {
			out = append(out, TermSuggestion{Kind: kind, Value: display[string(kind)+key], Count: c})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Value < out[j].Value
	})
	return out
}
