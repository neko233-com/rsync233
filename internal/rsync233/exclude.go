package rsync233

import (
	"path"
	"strings"
)

type Excluder struct {
	patterns []string
}

type Filter struct {
	rules []FilterRule
}

type FilterRule struct {
	Include bool
	Pattern string
}

func NewExcluder(patterns []string) Excluder {
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(filepathToSlash(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return Excluder{patterns: out}
}

func NewFilter(includes, excludes []string) Filter {
	rules := make([]FilterRule, 0, len(includes)+len(excludes))
	for _, p := range includes {
		p = strings.TrimSpace(filepathToSlash(p))
		if p != "" {
			rules = append(rules, FilterRule{Include: true, Pattern: p})
		}
	}
	for _, p := range excludes {
		p = strings.TrimSpace(filepathToSlash(p))
		if p != "" {
			rules = append(rules, FilterRule{Include: false, Pattern: p})
		}
	}
	return NewOrderedFilter(rules)
}

func NewOrderedFilter(rules []FilterRule) Filter {
	out := make([]FilterRule, 0, len(rules))
	for _, r := range rules {
		p := strings.TrimSpace(filepathToSlash(r.Pattern))
		if p != "" {
			r.Pattern = p
			out = append(out, r)
		}
	}
	return Filter{rules: out}
}

func (f Filter) Exclude(rel string, isDir bool) bool {
	for _, r := range f.rules {
		if (Excluder{patterns: []string{r.Pattern}}).Match(rel, isDir) {
			return !r.Include
		}
	}
	return false
}

func (e Excluder) Match(rel string, isDir bool) bool {
	rel = normalizeRel(rel)
	if rel == "." {
		return false
	}
	for _, p := range e.patterns {
		dirOnly := strings.HasSuffix(p, "/")
		pat := strings.TrimSuffix(p, "/")
		if dirOnly && !isDir {
			continue
		}
		if strings.HasPrefix(pat, "/") {
			pat = strings.TrimPrefix(pat, "/")
			if globMatch(pat, rel) {
				return true
			}
			continue
		}
		if globMatch(pat, rel) || globMatch(pat, path.Base(rel)) {
			return true
		}
		if strings.HasSuffix(pat, "/**") && strings.HasPrefix(rel, strings.TrimSuffix(pat, "/**")+"/") {
			return true
		}
	}
	return false
}

func globMatch(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}

func filepathToSlash(v string) string {
	return strings.ReplaceAll(v, "\\", "/")
}
