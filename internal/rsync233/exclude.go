package rsync233

import (
	"path"
	"strings"
)

type Excluder struct {
	patterns []string
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
