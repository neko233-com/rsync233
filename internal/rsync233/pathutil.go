package rsync233

import (
	"path"
	"path/filepath"
	"strings"
)

func joinPath(remote bool, elems ...string) string {
	if remote {
		return path.Clean(path.Join(elems...))
	}
	return filepath.Clean(filepath.Join(elems...))
}

func relPath(remote bool, base, target string) (string, error) {
	if remote {
		base = path.Clean(base)
		target = path.Clean(target)
		if base == target {
			return ".", nil
		}
		prefix := strings.TrimSuffix(base, "/") + "/"
		if strings.HasPrefix(target, prefix) {
			return strings.TrimPrefix(target, prefix), nil
		}
		return "", filepath.ErrBadPattern
	}
	rel, err := filepath.Rel(filepath.Clean(base), filepath.Clean(target))
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func normalizeRel(rel string) string {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	if rel == "" {
		return "."
	}
	return rel
}

func endpointBase(p string) string {
	return path.Base(filepathToSlash(p))
}
